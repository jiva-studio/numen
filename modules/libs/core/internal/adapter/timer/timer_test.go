package timer_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/timer"
)

func TestAWaitEndsWithItsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	start := time.Now()
	if err := timer.Wait(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Error("the wait did not end with the context")
	}
}

func TestAWaitOfNothingReportsTheContext(t *testing.T) {
	if err := timer.Wait(t.Context(), 0); err != nil {
		t.Errorf("got %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := timer.Wait(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}

func TestAWaitElapses(t *testing.T) {
	if err := timer.Wait(t.Context(), time.Millisecond); err != nil {
		t.Errorf("got %v", err)
	}
}

func TestAfterRunsFunction(t *testing.T) {
	var ran atomic.Bool
	stop := timer.After(time.Millisecond, func() {
		ran.Store(true)
	})
	defer stop()
	deadline := time.Now().Add(5 * time.Second)
	for !ran.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !ran.Load() {
		t.Fatal("the function did not run")
	}
}

func TestAfterStopPreventsExecution(t *testing.T) {
	var ran atomic.Bool
	stop := timer.After(50*time.Millisecond, func() {
		ran.Store(true)
	})
	if !stop() {
		t.Fatal("timer stop returned false")
	}
	time.Sleep(100 * time.Millisecond)
	if ran.Load() {
		t.Fatal("the function ran after being stopped")
	}
}
