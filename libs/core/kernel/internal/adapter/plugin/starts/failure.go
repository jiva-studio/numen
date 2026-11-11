// Package starts keeps why the start of a plugin failed.
package starts

import (
	"sync"
	"time"
)

// Failure keeps why the last start of a plugin failed and when.
type Failure struct {
	mu       sync.Mutex
	cause    error
	failedAt time.Time
}

// Record keeps why the last start failed and when, and reports whether the plugin had no failure before.
func (f *Failure) Record(now time.Time, cause error) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	isNew := f.cause == nil
	f.cause = cause
	f.failedAt = now
	return isNew
}

// Clear forgets the failure of the last start, and reports whether there was one.
func (f *Failure) Clear() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	hadFailure := f.cause != nil
	f.cause = nil
	f.failedAt = time.Time{}
	return hadFailure
}

// GetFailedAt returns when the last start failed, and the zero time while the plugin has no failure.
func (f *Failure) GetFailedAt() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.failedAt
}

// GetCause returns why the last start failed, and nil while it did not.
func (f *Failure) GetCause() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cause
}

// IsFailed reports whether the last start of the plugin failed.
func (f *Failure) IsFailed() bool {
	return f.GetCause() != nil
}
