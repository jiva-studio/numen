package lifecycle

import (
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

var (
	ErrActivationBackoff = errors.New("plugin activation is backing off after a failure")
	ErrPluginFailed      = errors.New("plugin failed to activate repeatedly")
	ErrActivationAborted = errors.New("plugin activation aborted")
)

// ActivationPolicy bounds how long one activation may take and how often a failing plugin is started again.
type ActivationPolicy struct {
	// Timeout is the deadline for a started process to print its handshake.
	Timeout time.Duration
	// BackoffBase is the wait after the first failure; each further failure doubles it.
	BackoffBase time.Duration
	// BackoffMax caps the wait between two attempts.
	BackoffMax time.Duration
	// MaxFailures is the number of consecutive failures after which the plugin is reported failed.
	MaxFailures int
}

// DefaultActivationPolicy returns the policy a new host applies.
func DefaultActivationPolicy() ActivationPolicy {
	return ActivationPolicy{
		Timeout:     30 * time.Second,
		BackoffBase: time.Second,
		BackoffMax:  time.Minute,
		MaxFailures: 5,
	}
}

func (p ActivationPolicy) GetBackoff(failures int) time.Duration {
	delay := p.BackoffBase
	for i := 1; i < failures && delay < p.BackoffMax; i++ {
		delay *= 2
	}
	if p.BackoffMax > 0 && delay > p.BackoffMax {
		return p.BackoffMax
	}
	return delay
}

// RecordActivationFailure counts a failed activation and reports whether the plugin has now failed for good.
func (s *Supervisor) RecordActivationFailure(now time.Time, policy ActivationPolicy, cause error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if policy.BackoffBase == 0 && policy.BackoffMax == 0 && policy.MaxFailures == 0 {
		policy = s.activationPolicy
	}
	s.failures++
	s.failure = cause
	s.retryAt = now.Add(policy.GetBackoff(s.failures))
	if (policy.MaxFailures > 0 && s.failures >= policy.MaxFailures) || errors.Is(cause, pluginport.ErrUnsupportedProtocol) {
		if !s.failed {
			s.failedAt = now
		}
		s.failed = true
	}
	return s.failed
}

// RecordActivationSuccess clears the failure count after a plugin activated.
func (s *Supervisor) RecordActivationSuccess() {
	s.ResetActivationFailures()
}

// ResetActivationFailures forgets earlier failed activations.
func (s *Supervisor) ResetActivationFailures() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures = 0
	s.failure = nil
	s.failed = false
	s.failedAt = time.Time{}
	s.retryAt = time.Time{}
}

// CheckActivationAllowed returns an error while the plugin has failed or is within its backoff.
func (s *Supervisor) CheckActivationAllowed(now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failed {
		return fmt.Errorf("%w: %w", ErrPluginFailed, s.failure)
	}
	if s.failures > 0 && now.Before(s.retryAt) {
		return fmt.Errorf("%w until %s: %w", ErrActivationBackoff, s.retryAt.Format(time.RFC3339), s.failure)
	}
	return nil
}

// GetFailedAt returns when the plugin was given up, and the zero time while it has not failed.
func (s *Supervisor) GetFailedAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failedAt
}

// GetActivationFailure returns the last activation error once the plugin has failed for good.
func (s *Supervisor) GetActivationFailure() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failed, s.failure
}

// DescribeFailure returns a short UI label for why a plugin failed to start. The error itself is the detail.
func DescribeFailure(cause error) string {
	switch {
	case errors.Is(cause, pluginport.ErrUnsupportedProtocol):
		return "Unsupported plugin protocol. Update the plugin or Numen"
	case errors.Is(cause, transport.ErrHandshakeTimeout):
		return "Timed out while starting"
	case errors.Is(cause, ErrMemoryExceeded):
		return "Exceeded its memory limit"
	default:
		return "Failed to start"
	}
}
