package ticker

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestTheFunctionRunsEachPeriodUntilStopped(t *testing.T) {
	var runs atomic.Int32
	reached := make(chan struct{})
	errLate := errors.New("the beds were not watered")
	stop := Start(time.Millisecond, func() error {
		switch runs.Add(1) {
		case 2:
			return errLate
		case 3:
			close(reached)
		}
		return nil
	})
	<-reached
	err := stop()
	after := runs.Load()
	time.Sleep(10 * time.Millisecond)

	if runs.Load() != after {
		t.Errorf("the function ran %d times after the stop", runs.Load()-after)
	}
	if !errors.Is(err, errLate) || !errors.Is(stop(), errLate) {
		t.Errorf("stop = %v, want the error of the last call that failed", err)
	}

	entered := make(chan struct{})
	errInFlight := errors.New("in-flight tick failed")
	var once atomic.Int32
	stopInFlight := Start(time.Millisecond, func() error {
		if once.Add(1) == 1 {
			close(entered)
		}
		time.Sleep(20 * time.Millisecond)
		return errInFlight
	})
	<-entered
	if got := stopInFlight(); !errors.Is(got, errInFlight) {
		t.Errorf("stopInFlight() = %v, want %v", got, errInFlight)
	}
}
