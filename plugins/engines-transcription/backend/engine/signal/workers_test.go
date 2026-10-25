package signal

import (
	"runtime"
	"testing"
)

func TestTheWorkersAskedForAreTheWorkersUsed(t *testing.T) {
	if got := GetWorkerCount(3, 100); got != 3 {
		t.Errorf("%d workers share the work, want 3", got)
	}
}

func TestNoMoreWorkersThanJobsShareTheWork(t *testing.T) {
	if got := GetWorkerCount(8, 2); got != 2 {
		t.Errorf("%d workers share 2 jobs, want 2", got)
	}
}

func TestAWorkerCountOfNoneConsumesOneWorkerPerCPU(t *testing.T) {
	want := runtime.GOMAXPROCS(0)
	if got := GetWorkerCount(0, want+10); got != want {
		t.Errorf("%d workers share the work, want %d", got, want)
	}
}

func TestAtLeastOneWorkerSharesTheWorkWhenThereAreNoJobs(t *testing.T) {
	if got := GetWorkerCount(4, 0); got != 1 {
		t.Errorf("%d workers share no jobs, want 1", got)
	}
}
