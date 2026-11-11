package process

import (
	"runtime"
	"testing"
)

func TestTheProcessCountsTheCoresOfTheMachine(t *testing.T) {
	if got := (Process{}).CountCPUs(); got != runtime.NumCPU() {
		t.Fatalf("cpus = %d, want %d", got, runtime.NumCPU())
	}
}

func TestSetMaxProcsLimitsTheThreadsThatRunGoCode(t *testing.T) {
	previous := runtime.GOMAXPROCS(0)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	Process{}.SetMaxProcs(1)

	if got := runtime.GOMAXPROCS(0); got != 1 {
		t.Fatalf("GOMAXPROCS = %d, want 1", got)
	}
}
