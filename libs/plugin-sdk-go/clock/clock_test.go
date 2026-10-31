package clock_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
)

func TestFakeReportsConfiguredTimeUntilMoved(t *testing.T) {
	start := time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC)
	fake := clock.NewFake(start)
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
