// Package clock provides the time sources of the core: a system clock for the
// composition root and a settable clock for tests.
package clock

import (
	"context"
	"sync"
	"time"
)

// System reads the time of the machine. The composition root passes it to the
// packages it constructs.
type System struct{}

// Now gets the current wall-clock time. This is the only read of the machine
// clock in the workspace; TestTheTypeCheckedWorkspaceObeysItsRules/TheMachineClockIsReadOnlyByAClockPortImplementation
// enforces it.
func (System) Now() time.Time {
	return time.Now()
}

// After returns a channel that receives the time once d has passed.
func (System) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// Wait waits for d, and returns early with the error of ctx where ctx ends first. A d of zero or less returns
// the error of ctx, which is nil while ctx is live.
func (System) Wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Fake is a clock that reports a configured time and moves only when told.
// It is safe for concurrent use.
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

func NewFake(start time.Time) *Fake {
	return &Fake{mu: sync.Mutex{}, now: start}
}

// Now gets the configured time.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Set moves the clock to t.
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = t
}

// Advance moves the clock forward by d and gets the new time.
func (f *Fake) Advance(d time.Duration) time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
	return f.now
}
