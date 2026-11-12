package plugin

import "github.com/jiva-studio/numen/libs/note/domain"

// UnavailableReason is why no plugin answers.
type UnavailableReason int

const (
	// NotInstalled is no installed plugin to answer.
	NotInstalled UnavailableReason = iota + 1
	// Disabled is every plugin that would answer switched off, by its setting or by safe mode.
	Disabled
	// Failed is a plugin that could not be started or reached.
	Failed
	// WithoutBackend is a plugin that declares no backend, so there is nothing to start.
	WithoutBackend
)

// UnavailableError is a plugin, or a capability, that no plugin answers for, and why. It is
// domain.ErrUnavailable.
type UnavailableError struct {
	Capability string
	PluginID   string
	Reason     UnavailableReason
	Err        error
}

// Error reports what is unavailable and why.
func (e *UnavailableError) Error() string {
	subject := "plugin " + e.PluginID
	if e.PluginID == "" {
		subject = "the plugin for " + e.Capability
	}
	var response string
	switch e.Reason {
	case NotInstalled:
		response = subject + " is not installed"
	case Disabled:
		response = subject + " is disabled"
	case WithoutBackend:
		response = subject + " has no backend"
	default:
		response = subject + " failed to start"
	}
	if e.Err != nil {
		return response + ": " + e.Err.Error()
	}
	return response
}

// Is reports that the error is domain.ErrUnavailable.
func (e *UnavailableError) Is(target error) bool {
	return target == domain.ErrUnavailable
}

// Unwrap returns the cause the host gave.
func (e *UnavailableError) Unwrap() error {
	return e.Err
}
