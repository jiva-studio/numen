package lifecycle

import (
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

var ErrActivationAborted = errors.New("plugin activation aborted")

// ActivationPolicy bounds how long one start of a plugin may take. A start that fails is not waited out: the
// next use of the plugin starts it again.
type ActivationPolicy struct {
	// Timeout is the deadline for a started process to print its handshake.
	Timeout time.Duration
}

// DefaultActivationPolicy returns the policy a new host applies.
func DefaultActivationPolicy() ActivationPolicy {
	return ActivationPolicy{Timeout: 30 * time.Second}
}

// RecordActivationFailure keeps why the last start failed and when, and reports whether the plugin had no
// failure before.
func (s *Supervisor) RecordActivationFailure(now time.Time, cause error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	isNew := s.failure == nil
	s.failure = cause
	s.failedAt = now
	return isNew
}

// ClearActivationFailure forgets the failure of the last start, and reports whether there was one.
func (s *Supervisor) ClearActivationFailure() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	hadFailure := s.failure != nil
	s.failure = nil
	s.failedAt = time.Time{}
	return hadFailure
}

// GetFailedAt returns when the last start failed, and the zero time while the plugin has no failure.
func (s *Supervisor) GetFailedAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failedAt
}

// GetActivationFailure returns why the last start failed, and nil while it did not.
func (s *Supervisor) GetActivationFailure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failure
}

// IsActivationFailed reports whether the last start of the plugin failed.
func (s *Supervisor) IsActivationFailed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failure != nil
}

// DescribeFailure returns a short UI label for why a plugin failed to start. The error itself is the detail.
func DescribeFailure(cause error) string {
	switch {
	case errors.Is(cause, pluginport.ErrUnsupportedProtocol):
		return "Unsupported plugin protocol. Update the plugin or Numen"
	case errors.Is(cause, transport.ErrHandshakeTimeout):
		return "Timed out while starting"
	default:
		return "Failed to start"
	}
}
