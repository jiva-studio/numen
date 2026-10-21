package pdf

import (
	"runtime"
	"testing"
)

func withWorkers(t *testing.T, n int) {
	t.Helper()
	workers.Lock()
	before := workers.size
	workers.Unlock()
	t.Cleanup(func() { SetWorkers(before) })
	SetWorkers(n)
}

func TestThePoolStartsWithTheWorkersItIsSetTo(t *testing.T) {
	withWorkers(t, minWorkers+3)
	if got, want := GetWorkers(), minWorkers+3; got != want {
		t.Errorf("the pool starts with %d workers, want %d", got, want)
	}
}

func TestThePoolStartsWithOneWorkerPerCPUWhenNoneIsSet(t *testing.T) {
	withWorkers(t, 0)
	if got, want := GetWorkers(), max(runtime.NumCPU(), minWorkers); got != want {
		t.Errorf("the pool starts with %d workers, want %d", got, want)
	}
}

func TestThePoolStartsWithAtLeastTheMinimumWhenFewerAreSet(t *testing.T) {
	withWorkers(t, 1)
	if got := GetWorkers(); got != 2 {
		t.Errorf("the pool starts with %d workers, want 2", got)
	}
}
