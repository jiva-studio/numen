package clock_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
)

func TestSystemReadsMachineTime(t *testing.T) {
	var system clock.System
	first := system.Now()
	second := system.Now()
	if second.Before(first) {
		t.Fatalf("second reading %v precedes first %v", second, first)
	}
	if first.Year() < 2025 {
		t.Fatalf("reading %v is not machine time", first)
	}
}

func TestSystemWaitReturnsTheErrorOfAContextThatEndsFirst(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := (clock.System{}).Wait(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Errorf("a wait on a canceled context returned %v", err)
	}
	if err := (clock.System{}).Wait(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("a wait of no time on a canceled context returned %v", err)
	}
}

func TestSystemWaitReturnsNilOnceTheTimeHasPassed(t *testing.T) {
	if err := (clock.System{}).Wait(t.Context(), time.Millisecond); err != nil {
		t.Errorf("a wait returned %v", err)
	}
	if err := (clock.System{}).Wait(t.Context(), 0); err != nil {
		t.Errorf("a wait of no time returned %v", err)
	}
}

func TestSystemAfterReceivesTheTimeOnceDHasPassed(t *testing.T) {
	select {
	case <-(clock.System{}).After(time.Millisecond):
	case <-time.After(time.Minute):
		t.Fatal("the channel received nothing")
	}
}
