// Package clocktest holds the clock plugin tests set and move by hand.
package clocktest

import (
	"sync"
	"time"
)

// Fake is a clock that reports the time Set gave it, the zero time before, and moves only when told.
// It is safe for concurrent use.
type Fake struct {
	mu  sync.Mutex
	now time.Time
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
