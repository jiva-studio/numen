package testsupport

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestManualClockReportsConfiguredTimeUntilMoved(t *testing.T) {
	start := time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC)
	fake := NewClockAt(start)
	if got := fake.Now(); !got.Equal(start) {
		t.Fatalf("Now() = %v, want %v", got, start)
	}
	if got := fake.Advance(time.Hour); !got.Equal(start.Add(time.Hour)) {
		t.Fatalf("Advance() = %v, want %v", got, start.Add(time.Hour))
	}
	later := start.Add(48 * time.Hour)
	fake.Set(later)
	if got := fake.Now(); !got.Equal(later) {
		t.Fatalf("Now() after Set = %v, want %v", got, later)
	}
}

func TestManualClockFiresAWaitOnlyOnceMovedPastItsEnd(t *testing.T) {
	fake := NewClock()
	fired := fake.After(time.Minute)
	fake.Advance(59 * time.Second)
	select {
	case <-fired:
		t.Fatal("the wait ended a second before its end")
	default:
	}
	fake.Advance(time.Second)
	select {
	case <-fired:
	default:
		t.Fatal("the wait did not end when the clock reached its end")
	}
}

func TestManualClockWaitEndsWithTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := NewClock().Wait(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait() = %v, want %v", err, context.Canceled)
	}
}
