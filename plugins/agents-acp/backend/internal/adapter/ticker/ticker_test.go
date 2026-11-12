package ticker

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestTheFunctionRunsEachPeriodUntilStopped(t *testing.T) {
	var runs atomic.Int32
	reached := make(chan struct{})
	stop := Start(time.Millisecond, func() {
		if runs.Add(1) == 3 {
			close(reached)
		}
	})
	<-reached
	stop()
	stop()
	after := runs.Load()
	time.Sleep(10 * time.Millisecond)

	if runs.Load() != after {
		t.Errorf("the function ran %d times after the stop", runs.Load()-after)
	}
}
