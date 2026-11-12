package testsupport

import (
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
)

// Epoch is the moment that the clock of a test reports.
func GetEpoch() time.Time { return time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC) }

// Machine reads the time of the machine. A test uses it where it compares a
// reading with the modification times of files or measures elapsed time.
func GetMachine() clock.System { return clock.System{} }

// NewClock creates a clock that reports Epoch until it is moved.
func NewClock() *ManualClock {
	return NewClockAt(GetEpoch())
}

// NewClockAt creates a clock that reports start until it is moved.
func NewClockAt(start time.Time) *ManualClock {
	return &ManualClock{now: start}
}

// ManualClock is a clock that reports a configured time and moves only when told.
// It is safe for concurrent use.
type ManualClock struct {
	mu  sync.Mutex
	now time.Time
}

// Now gets the configured time.
func (f *ManualClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Set moves the clock to t.
func (f *ManualClock) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = t
}

// Advance moves the clock forward by d and gets the new time.
func (f *ManualClock) Advance(d time.Duration) time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
	return f.now
}
