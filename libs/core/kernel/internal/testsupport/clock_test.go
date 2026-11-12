package testsupport

import (
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
