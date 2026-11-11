package pdf

import (
	"runtime"
	"testing"
)

func TestThePoolStartsWithTheWorkersItIsSetTo(t *testing.T) {
	workers := NewWorkers(Options{Report: failOnReport(t), CompilationCache: nil, AfterFunc: nil, Workers: minWorkers + 3, IdleWorkers: 0})
	if got, want := workers.GetWorkers(), minWorkers+3; got != want {
		t.Errorf("the pool starts with %d workers, want %d", got, want)
	}
}

func TestThePoolStartsWithOneWorkerPerCPUWhenNoneIsSet(t *testing.T) {
	workers := NewWorkers(Options{Report: failOnReport(t), CompilationCache: nil, AfterFunc: nil, Workers: 0, IdleWorkers: 0})
	if got, want := workers.GetWorkers(), max(runtime.NumCPU(), minWorkers); got != want {
		t.Errorf("the pool starts with %d workers, want %d", got, want)
	}
}

func TestThePoolStartsWithAtLeastTheMinimumWhenFewerAreSet(t *testing.T) {
	workers := NewWorkers(Options{Report: failOnReport(t), CompilationCache: nil, AfterFunc: nil, Workers: 1, IdleWorkers: 0})
	if got := workers.GetWorkers(); got != 2 {
		t.Errorf("the pool starts with %d workers, want 2", got)
	}
}
