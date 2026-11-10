package lifecycle

import (
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

var ErrActivationAborted = errors.New("plugin activation aborted")

// ActivationPolicy bounds how long one start of a plugin may take. A start that fails is not waited out: the
// next use of the plugin starts it again.
type ActivationPolicy struct {
	Timeout time.Duration
}

// DefaultActivationPolicy returns the policy a new host applies.
func DefaultActivationPolicy() ActivationPolicy {
	return ActivationPolicy{Timeout: 30 * time.Second}
}

// StartFailure keeps why the last start of a plugin failed and when.
type StartFailure struct {
	mu       sync.Mutex
	cause    error
	failedAt time.Time
}

// Record keeps why the last start failed and when, and reports whether the plugin had no failure before.
func (f *StartFailure) Record(now time.Time, cause error) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	isNew := f.cause == nil
	f.cause = cause
	f.failedAt = now
	return isNew
}

// Clear forgets the failure of the last start, and reports whether there was one.
func (f *StartFailure) Clear() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	hadFailure := f.cause != nil
	f.cause = nil
	f.failedAt = time.Time{}
	return hadFailure
}

// GetFailedAt returns when the last start failed, and the zero time while the plugin has no failure.
func (f *StartFailure) GetFailedAt() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.failedAt
}

// GetCause returns why the last start failed, and nil while it did not.
func (f *StartFailure) GetCause() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cause
}

// IsFailed reports whether the last start of the plugin failed.
func (f *StartFailure) IsFailed() bool {
	return f.GetCause() != nil
}

// DescribeFailure returns a short UI label for why a plugin failed to start. The error itself is the detail.
func DescribeFailure(cause error) string {
	switch {
	case errors.Is(cause, manifest.ErrUnsupportedProtocol):
		return "Unsupported plugin protocol. Update the plugin or Numen"
	case errors.Is(cause, transport.ErrHandshakeTimeout):
		return "Timed out while starting"
	default:
		return "Failed to start"
	}
}
