package process

import (
	"runtime"
	"testing"
)

func TestSetMaxProcsLimitsTheThreadsThatRunGoCode(t *testing.T) {
	previous := runtime.GOMAXPROCS(0)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	Process{}.SetMaxProcs(1)
	if got := runtime.GOMAXPROCS(0); got != 1 {
		t.Fatalf("GOMAXPROCS = %d, want 1", got)
	}
	Process{}.SetMaxProcs(2)
	if got := runtime.GOMAXPROCS(0); got != 2 {
		t.Fatalf("GOMAXPROCS = %d, want 2", got)
	}
}
