package pdf

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestTheIdleWorkersFitTheAvailableMemory(t *testing.T) {
	const mb = int64(1 << 20)
	cases := []struct {
		name      string
		planned   int
		available int64
		want      int
	}{
		{"a small laptop is capped by memory", 4, 2048 * mb, 3},
		{"a desktop is capped by the plan", 6, 16 * 1024 * mb, 6},
		{"a server is capped by the plan", 32, 64 * 1024 * mb, 32},
		{"unknown memory keeps none", 8, 0, 0},
		{"no plan keeps none", 0, 16 * 1024 * mb, 0},
		{"negative plan keeps none", -1, 2048 * mb, 0},
		{"negative memory keeps none", 4, -1, 0},
		{"one planned worker with minimal fitting memory", 1, 700 * mb, 1},
		{"one planned worker with 1 byte of memory", 1, 1, 0},
	}
	for _, c := range cases {
		if got := ComputeIdleWorkers(c.planned, c.available); got != c.want {
			t.Errorf("%s: %d idle workers, want %d", c.name, got, c.want)
		}
	}
	if got := ComputeIdleWorkers(4, 2048*mb); got < 1 || got >= 4 {
		t.Errorf("a small laptop keeps %d idle workers, want fewer than planned but some", got)
	}
}

func startIdlePool(t *testing.T, idle int, period time.Duration) *Workers {
	t.Helper()
	workers := NewWorkers(Options{
		Report: failOnReport(t), CompilationCache: newCompilation(t), Workers: 0, IdleWorkers: idle,
		AfterFunc: func(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) },
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
	workers := NewWorkers(Options{Report: failOnReport(t), CompilationCache: nil, AfterFunc: nil, Workers: 0, IdleWorkers: -1})
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
