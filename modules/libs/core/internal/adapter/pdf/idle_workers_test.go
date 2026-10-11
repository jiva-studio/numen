package pdf

import (
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
		{"a small laptop is capped by memory", 4, 2048 * mb, int(2048 * mb / 100 * idleMemoryPercent / workerBytes)},
		{"a desktop is capped by the plan", 6, 16 * 1024 * mb, 6},
		{"a server is capped by the plan", 32, 64 * 1024 * mb, 32},
		{"unknown memory keeps none", 8, 0, 0},
		{"no plan keeps none", 0, 16 * 1024 * mb, 0},
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
	before, beforePeriod := workers.idle, idleRelease
	workers.Unlock()
	t.Cleanup(func() {
		releasePool()
		SetIdleWorkers(before)
		workers.Lock()
		idleRelease = beforePeriod
		workers.Unlock()
	})
	releasePool()
	SetIdleWorkers(idle)
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
	first.close()
	second, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer second.close()
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
	first.close()
	second, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer second.close()
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
	doc.close()
	deadline := time.Now().Add(5 * time.Second)
	for hasPool() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hasPool() {
		t.Fatal("the pool was not released after the idle period")
	}
	again, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	again.close()
}

func TestReleasingThePoolClosesEveryWorker(t *testing.T) {
	startIdlePool(t, 2, time.Hour)
	doc, err := openPath("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	doc.close()
	releasePool()
	if hasPool() {
		t.Fatal("the pool is still held after it was released")
	}
}
