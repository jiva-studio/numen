package pdf

import (
	"context"
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
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

// startIdlePool releases any running pool and sets the idle count and period
// the next pool starts with.
func startIdlePool(t *testing.T, idle int, period time.Duration) {
	t.Helper()
	workers.Lock()
	before, beforePeriod, beforeAfter := workers.idle, idleRelease, workers.afterFunc
	workers.Unlock()
	t.Cleanup(func() {
		releasePool()
		SetIdleWorkers(before)
		workers.Lock()
		idleRelease = beforePeriod
		workers.afterFunc = beforeAfter
		workers.Unlock()
	})
	releasePool()
	SetIdleWorkers(idle)
	SetAfterFunc(func(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) })
	workers.Lock()
	idleRelease = period
	workers.Unlock()
}

func TestAnIdleWorkerIsReusedByTheNextDocument(t *testing.T) {
	startIdlePool(t, 2, time.Hour)
	first, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	worker := first.worker.GetImplementation()
	cleanup.Close(t, first)
	second, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, second)
	if second.worker.GetImplementation() != worker {
		t.Error("the second document started a new worker, want the first one's")
	}
	if second.pages == 0 {
		t.Error("the reused worker read no pages")
	}
}

func TestWithNoIdleCountEveryDocumentStartsAWorker(t *testing.T) {
	startIdlePool(t, 0, time.Hour)
	first, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	worker := first.worker.GetImplementation()
	cleanup.Close(t, first)
	second, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, second)
	if second.worker.GetImplementation() == worker {
		t.Error("the second document reused a worker, want a new one")
	}
}

func hasPool() bool {
	workers.Lock()
	defer workers.Unlock()
	return workers.pool != nil
}

func TestTheWorkersAreReleasedAfterTheIdlePeriod(t *testing.T) {
	startIdlePool(t, 2, 50*time.Millisecond)
	doc, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	if !hasPool() {
		t.Fatal("the pool was released while a document was open")
	}
	cleanup.Close(t, doc)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for hasPool() && ctx.Err() == nil {
		time.Sleep(10 * time.Millisecond)
	}
	if hasPool() {
		t.Fatal("the pool was not released after the idle period")
	}
	again, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	cleanup.Close(t, again)
}

func TestReleasingThePoolClosesEveryWorker(t *testing.T) {
	startIdlePool(t, 2, time.Hour)
	doc, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	cleanup.Close(t, doc)
	releasePool()
	if hasPool() {
		t.Fatal("the pool is still held after it was released")
	}
}

func TestAnOpenRacingTheIdleReleaseStartsAFreshPool(t *testing.T) {
	startIdlePool(t, 2, time.Microsecond)
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 150 {
				doc, err := openPath("testdata/tiny.pdf")
				if err != nil {
					errs <- err
					return
				}
				cleanup.Close(t, doc)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestAReleaseCannotCloseAPoolADocumentHasAcquired(t *testing.T) {
	startIdlePool(t, 2, time.Hour)
	p, err := acquirePool()
	if err != nil {
		t.Fatal(err)
	}
	releasePool()
	if !hasPool() {
		markDocumentClosed()
		t.Fatal("the pool was released while a document had acquired it")
	}
	worker, err := p.GetInstance(time.Minute)
	if err != nil {
		markDocumentClosed()
		t.Fatal(err)
	}
	if err := putWorker(worker); err != nil {
		t.Fatal(err)
	}
}

func openPath(path string) (*document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}
	return openReader(file)
}

func TestSetIdleWorkersNegativeKeepsNone(t *testing.T) {
	workers.Lock()
	before := workers.idle
	workers.Unlock()
	t.Cleanup(func() { SetIdleWorkers(before) })

	SetIdleWorkers(-1)
	workers.Lock()
	idle := workers.idle
	workers.Unlock()
	if idle != 0 {
		t.Errorf("SetIdleWorkers(-1) set idle to %d, want 0", idle)
	}
}

func TestAcquiringDocumentCancelsPendingIdleRelease(t *testing.T) {
	startIdlePool(t, 2, time.Hour)
	doc, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	cleanup.Close(t, doc)

	workers.Lock()
	hasTimer := workers.release != nil
	workers.Unlock()
	if !hasTimer {
		t.Fatal("expected pending idle release timer after closing document")
	}

	again, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, again)

	workers.Lock()
	timerStopped := workers.release == nil
	workers.Unlock()
	if !timerStopped {
		t.Error("expected release timer to be cancelled and cleared upon acquiring document")
	}
}
