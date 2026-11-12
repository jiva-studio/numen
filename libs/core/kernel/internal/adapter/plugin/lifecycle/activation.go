package lifecycle

import (
	"errors"
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
