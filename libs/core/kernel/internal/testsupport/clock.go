package testsupport

import (
	"context"
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

// ManualClock is a clock that reports a configured time and moves only when told. A wait on it ends when
// the clock is moved to or past its end. It is safe for concurrent use.
type ManualClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []manualWaiter
}

type manualWaiter struct {
	end   time.Time
	fired chan time.Time
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
	f.fireDue()
}

// Advance moves the clock forward by d and gets the new time.
func (f *ManualClock) Advance(d time.Duration) time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
	f.fireDue()
	return f.now
}

// After returns a channel that receives the time once the clock has moved d past the time it reads now.
func (f *ManualClock) After(d time.Duration) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	fired := make(chan time.Time, 1)
	f.waiters = append(f.waiters, manualWaiter{end: f.now.Add(d), fired: fired})
	f.fireDue()
	return fired
}

// Wait waits until the clock has moved d past the time it reads now, and returns early with the error of ctx
// where ctx ends first. A d of zero or less returns the error of ctx, as the system clock does.
func (f *ManualClock) Wait(ctx context.Context, d time.Duration) error {
	if d <= 0 || ctx.Err() != nil {
		return ctx.Err()
	}
	fired := f.After(d)
	select {
	case <-ctx.Done():
		f.drop(fired)
		return ctx.Err()
	case <-fired:
		return nil
	}
}

func (f *ManualClock) drop(fired <-chan time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	pending := f.waiters[:0]
	for _, one := range f.waiters {
		if one.fired != fired {
			pending = append(pending, one)
		}
	}
	f.waiters = pending
}

func (f *ManualClock) fireDue() {
	pending := f.waiters[:0]
	for _, one := range f.waiters {
		if one.end.After(f.now) {
			pending = append(pending, one)
			continue
		}
		one.fired <- f.now
	}
	f.waiters = pending
}
