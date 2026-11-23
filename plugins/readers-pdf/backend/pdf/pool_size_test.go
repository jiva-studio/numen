package pdf

import (
	"runtime"
	"testing"
)

const setWorkerCount = 5

func TestThePoolStartsWithTheWorkersItIsSetTo(t *testing.T) {
	workers := NewWorkers(Options{Report: failOnReport(t), CompilationCache: nil, StartTimer: nil, Workers: setWorkerCount, IdleWorkers: 0})
	if got := workers.GetWorkers(); got != setWorkerCount {
		t.Errorf("the pool starts with %d workers, want %d", got, setWorkerCount)
	}
}

func TestThePoolStartsWithOneWorkerPerCPUWhenNoneIsSet(t *testing.T) {
	workers := NewWorkers(Options{Report: failOnReport(t), CompilationCache: nil, StartTimer: nil, Workers: 0, IdleWorkers: 0})
	if got, want := workers.GetWorkers(), runtime.NumCPU(); got != want {
		t.Errorf("the pool starts with %d workers, want %d", got, want)
	}
}

func TestThePoolStartsWithOneWorkerWhenOneIsSet(t *testing.T) {
	workers := NewWorkers(Options{Report: failOnReport(t), CompilationCache: nil, StartTimer: nil, Workers: 1, IdleWorkers: 0})
	if got := workers.GetWorkers(); got != 1 {
		t.Errorf("the pool starts with %d workers, want 1", got)
	}
}
