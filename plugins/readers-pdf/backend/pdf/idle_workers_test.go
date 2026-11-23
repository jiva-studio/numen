package pdf

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/webassembly"
)

func startIdlePool(t *testing.T, idle int, period time.Duration) *Workers {
	t.Helper()
	workers := NewWorkers(Options{
		Report: failOnReport(t), CompilationCache: newCompilation(t), Workers: 0, IdleWorkers: idle,
		StartTimer: func(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) },
	})
	t.Cleanup(workers.releasePool)
	workers.idleRelease = period
	return workers
}

func TestAnIdleWorkerIsReusedByTheNextDocument(t *testing.T) {
	workers := startIdlePool(t, 2, time.Hour)
	first, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	worker := first.worker.GetImplementation()
	closeAndCheck(t, first)
	second, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer closeAndCheck(t, second)
	if second.worker.GetImplementation() != worker {
		t.Error("the second document started a new worker, want the first one's")
	}
	if second.pages == 0 {
		t.Error("the reused worker read no pages")
	}
}

func TestWithNoIdleCountEveryDocumentStartsAWorker(t *testing.T) {
	workers := startIdlePool(t, 0, time.Hour)
	first, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	worker := first.worker.GetImplementation()
	closeAndCheck(t, first)
	second, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer closeAndCheck(t, second)
	if second.worker.GetImplementation() == worker {
		t.Error("the second document reused a worker, want a new one")
	}
}

func hasPool(workers *Workers) bool {
	workers.mu.Lock()
	defer workers.mu.Unlock()
	return workers.pool != nil
}

func TestTheWorkersAreReleasedAfterTheIdlePeriod(t *testing.T) {
	workers := startIdlePool(t, 2, 50*time.Millisecond)
	doc, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	if !hasPool(workers) {
		t.Fatal("the pool was released while a document was open")
	}
	closeAndCheck(t, doc)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for hasPool(workers) && ctx.Err() == nil {
		time.Sleep(10 * time.Millisecond)
	}
	if hasPool(workers) {
		t.Fatal("the pool was not released after the idle period")
	}
	again, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	closeAndCheck(t, again)
}

func TestReleasingThePoolClosesEveryWorker(t *testing.T) {
	workers := startIdlePool(t, 2, time.Hour)
	doc, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	closeAndCheck(t, doc)
	workers.releasePool()
	if hasPool(workers) {
		t.Fatal("the pool is still held after it was released")
	}
	if workers.release != nil {
		t.Fatal("the idle release timer is still held after the pool was released")
	}
}

func TestAnOpenRacingTheIdleReleaseStartsAFreshPool(t *testing.T) {
	workers := startIdlePool(t, 2, time.Microsecond)
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Go(func() {
			for range 150 {
				doc, err := openPath(workers, "testdata/tiny.pdf")
				if err != nil {
					errs <- err
					return
				}
				closeAndCheck(t, doc)
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestAReleaseCannotCloseAPoolADocumentHasAcquired(t *testing.T) {
	workers := startIdlePool(t, 2, time.Hour)
	p, err := workers.acquirePool()
	if err != nil {
		t.Fatal(err)
	}
	workers.releasePool()
	if !hasPool(workers) {
		workers.markDocumentClosed()
		t.Fatal("the pool was released while a document had acquired it")
	}
	worker, err := p.GetInstance(time.Minute)
	if err != nil {
		workers.markDocumentClosed()
		t.Fatal(err)
	}
	if err := workers.putWorker(worker); err != nil {
		t.Fatal(err)
	}
}

func openPath(workers *Workers, path string) (*document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}
	return workers.openReader(file)
}

func TestNegativeIdleWorkersKeepNone(t *testing.T) {
	workers := NewWorkers(Options{Report: failOnReport(t), CompilationCache: nil, StartTimer: nil, Workers: 0, IdleWorkers: -1})
	workers.mu.Lock()
	idle := workers.idle
	workers.mu.Unlock()
	if idle != 0 {
		t.Errorf("IdleWorkers of -1 set idle to %d, want 0", idle)
	}
}

func TestAcquiringDocumentCancelsPendingIdleRelease(t *testing.T) {
	workers := startIdlePool(t, 2, time.Hour)
	doc, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	closeAndCheck(t, doc)

	workers.mu.Lock()
	hasTimer := workers.release != nil
	workers.mu.Unlock()
	if !hasTimer {
		t.Fatal("expected pending idle release timer after closing document")
	}

	again, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer closeAndCheck(t, again)

	workers.mu.Lock()
	timerStopped := workers.release == nil
	workers.mu.Unlock()
	if !timerStopped {
		t.Error("expected release timer to be canceled and cleared upon acquiring document")
	}
}

func getIdleCount(workers *Workers) int {
	workers.mu.Lock()
	defer workers.mu.Unlock()
	return workers.idle
}

func TestANewIdleCountClosesThePoolOnceTheLastDocumentCloses(t *testing.T) {
	workers := startIdlePool(t, 1, time.Hour)
	doc, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	workers.SetWorkers(0, 3)
	if !hasPool(workers) {
		closeAndCheck(t, doc)
		t.Fatal("the pool was closed while a document was open on it")
	}
	closeAndCheck(t, doc)
	if hasPool(workers) {
		t.Fatal("the pool of the old idle count is kept after the last document closed")
	}
	if got := getIdleCount(workers); got != 3 {
		t.Errorf("the pool keeps %d idle workers, want 3", got)
	}
	again, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	closeAndCheck(t, again)
	if !hasPool(workers) {
		t.Error("the pool of the new idle count closed with its document")
	}
}

func TestANewIdleCountClosesAPoolNoDocumentIsOpenOn(t *testing.T) {
	workers := startIdlePool(t, 2, time.Hour)
	doc, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	closeAndCheck(t, doc)
	workers.SetWorkers(0, 1)
	if hasPool(workers) {
		t.Fatal("the idle pool of the old count is kept")
	}
}

func TestTheSameIdleCountKeepsThePool(t *testing.T) {
	workers := startIdlePool(t, 2, time.Hour)
	doc, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	closeAndCheck(t, doc)
	workers.SetWorkers(0, 2)
	if !hasPool(workers) {
		t.Fatal("the pool was closed for an idle count it already had")
	}
}

func TestANewIdleCountDoesNotWaitForThePoolToStart(t *testing.T) {
	workers := startIdlePool(t, 1, time.Hour)
	isStarting := make(chan struct{})
	finishStart := make(chan struct{})
	workers.startPool = func(config webassembly.Config) (pdfium.Pool, error) {
		close(isStarting)
		<-finishStart
		return webassembly.Init(config)
	}
	opened := make(chan *document, 1)
	go func() {
		doc, err := openPath(workers, "testdata/tiny.pdf")
		if err != nil {
			t.Error(err)
		}
		opened <- doc
	}()
	<-isStarting
	isSet := make(chan struct{})
	go func() {
		workers.SetWorkers(0, 3)
		close(isSet)
	}()
	select {
	case <-isSet:
	case <-time.After(5 * time.Second):
		close(finishStart)
		t.Fatal("setting the idle count waited for the pool to start")
	}
	close(finishStart)
	doc := <-opened
	if doc == nil {
		return
	}
	closeAndCheck(t, doc)
	if hasPool(workers) {
		t.Fatal("the pool started with the old idle count is kept after its document closed")
	}
	if got := getIdleCount(workers); got != 3 {
		t.Errorf("the pool keeps %d idle workers, want 3", got)
	}
}

func TestANewWorkerCountStartsThePoolOfThatSize(t *testing.T) {
	workers := startIdlePool(t, 2, time.Hour)
	workers.SetWorkers(oldWorkerCount, 2)
	doc, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	closeAndCheck(t, doc)
	var started []int
	workers.startPool = func(config webassembly.Config) (pdfium.Pool, error) {
		started = append(started, config.MaxTotal)
		return webassembly.Init(config)
	}
	workers.SetWorkers(newWorkerCount, 2)
	if hasPool(workers) {
		t.Fatal("the idle pool of the old worker count is kept")
	}
	again, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	closeAndCheck(t, again)
	if len(started) != 1 || started[0] != newWorkerCount {
		t.Errorf("the pools started with %v workers, want [%d]", started, newWorkerCount)
	}
}

func TestANewWorkerCountClosesThePoolOnceTheLastDocumentCloses(t *testing.T) {
	workers := startIdlePool(t, 2, time.Hour)
	workers.SetWorkers(oldWorkerCount, 2)
	doc, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	workers.SetWorkers(newWorkerCount, 2)
	if !hasPool(workers) {
		closeAndCheck(t, doc)
		t.Fatal("the pool was closed while a document was open on it")
	}
	closeAndCheck(t, doc)
	if hasPool(workers) {
		t.Fatal("the pool of the old worker count is kept after the last document closed")
	}
	if got := workers.GetWorkers(); got != newWorkerCount {
		t.Errorf("the pool starts with %d workers, want %d", got, newWorkerCount)
	}
}

const (
	oldWorkerCount = 2
	newWorkerCount = 3

	heldWorkerWait  = 200 * time.Millisecond
	freedWorkerWait = 5 * time.Second
)

func TestASecondDocumentWaitsForTheOnlyWorker(t *testing.T) {
	workers := startIdlePool(t, 1, time.Hour)
	workers.SetWorkers(1, 1)
	first, err := openPath(workers, "testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	opened := make(chan *document, 1)
	go func() {
		doc, err := openPath(workers, "testdata/tiny.pdf")
		if err != nil {
			t.Error(err)
		}
		opened <- doc
	}()
	select {
	case doc := <-opened:
		closeAndCheck(t, first)
		if doc != nil {
			closeAndCheck(t, doc)
		}
		t.Fatal("the second document opened while the only worker held the first")
	case <-time.After(heldWorkerWait):
	}
	closeAndCheck(t, first)
	select {
	case doc := <-opened:
		if doc == nil {
			return
		}
		if doc.pages == 0 {
			t.Error("the second document read no pages")
		}
		closeAndCheck(t, doc)
	case <-time.After(freedWorkerWait):
		t.Fatal("the second document did not open after the first closed")
	}
}
