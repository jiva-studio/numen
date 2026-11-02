package pdf

import (
	"runtime"
	"testing"
)

func TestThePoolStartsWithTheWorkersItIsSetTo(t *testing.T) {
	workers := NewWorkers()
	workers.SetWorkers(minWorkers + 3)
	if got, want := workers.GetWorkers(), minWorkers+3; got != want {
		t.Errorf("the pool starts with %d workers, want %d", got, want)
	}
}

func TestThePoolStartsWithOneWorkerPerCPUWhenNoneIsSet(t *testing.T) {
	workers := NewWorkers()
	workers.SetWorkers(0)
	if got, want := workers.GetWorkers(), max(runtime.NumCPU(), minWorkers); got != want {
		t.Errorf("the pool starts with %d workers, want %d", got, want)
	}
}

func TestThePoolStartsWithAtLeastTheMinimumWhenFewerAreSet(t *testing.T) {
	workers := NewWorkers()
	workers.SetWorkers(1)
	if got := workers.GetWorkers(); got != 2 {
		t.Errorf("the pool starts with %d workers, want 2", got)
	}
}
