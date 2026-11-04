package signal

import (
	"runtime"
	"testing"
)

func TestGetWorkerCount_UsesRequestedCount(t *testing.T) {
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
	prev := runtime.GOMAXPROCS(4)
	defer runtime.GOMAXPROCS(prev)

	if got := GetWorkerCount(0, 10); got != 4 {
		t.Errorf("%d workers share the work for 0, want 4", got)
	}
	if got := runtime.GOMAXPROCS(0); got != 4 {
		t.Errorf("GOMAXPROCS changed to %d, want 4", got)
	}
	if got := GetWorkerCount(-1, 10); got != 4 {
		t.Errorf("%d workers share the work for -1, want 4", got)
	}
	if got := GetWorkerCount(1, 10); got != 1 {
		t.Errorf("%d workers share the work for 1, want 1", got)
	}
}

func TestAtLeastOneWorkerSharesTheWorkWhenThereAreNoJobs(t *testing.T) {
	if got := GetWorkerCount(4, 0); got != 1 {
		t.Errorf("%d workers share no jobs, want 1", got)
	}
}
