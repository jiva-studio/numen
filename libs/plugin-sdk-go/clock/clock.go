// Package clock provides the time source plugins receive through their
// constructors: a system clock for the plugin entry point and a settable clock
// for tests.
package clock

import (
	"sync"
	"time"
)

// System reads the time of the machine. The plugin entry point passes it to
// the packages it constructs.
type System struct{}

// Now gets the current wall-clock time. This is the only read of the machine
// clock in the workspace; TestTheMachineClockIsReadOnlyByAClockPortImplementation
// enforces it.
func (System) Now() time.Time {
	return time.Now()
}

// Fake is a clock that reports a configured time and moves only when told.
// It is safe for concurrent use.
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

// NewFake creates a Fake that reports start.
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
