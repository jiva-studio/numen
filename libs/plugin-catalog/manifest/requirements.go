package manifest

import (
	"errors"
	"fmt"
	"slices"
)

// IncompatibleReason is why an application cannot run a plugin.
type IncompatibleReason string

// The reasons an application cannot run a plugin.
const (
	// ReasonTarget is a plugin that does not run in the window the application opens.
	ReasonTarget IncompatibleReason = "target"
	// ReasonProtocol is a plugin that uses a newer plugin protocol than the application.
	ReasonProtocol IncompatibleReason = "protocol"
)

// ErrIncompatible is a plugin that an application cannot run.
var ErrIncompatible = errors.New("plugin is incompatible with the application")

// IncompatibleError is a plugin that an application cannot run, and why.
type IncompatibleError struct {
	Reason IncompatibleReason
	Detail string
}

// Error states the reason.
func (e *IncompatibleError) Error() string {
	return fmt.Sprintf("%s: %s: %s", ErrIncompatible, e.Reason, e.Detail)
}

// Is reports that the error is ErrIncompatible.
func (e *IncompatibleError) Is(target error) bool {
	return target == ErrIncompatible
}

// Environment is the application a plugin is checked against.
type Environment struct {
	Target string
}

// Requirements are what a plugin requires of the application that runs it.
type Requirements struct {
	Targets  []string
	Protocol int
}

// GetRequirements returns what the plugin requires of the application.
func (m Manifest) GetRequirements() Requirements {
	return Requirements{Targets: slices.Clone(m.Targets), Protocol: m.GetProtocol()}
}

// Check returns an *IncompatibleError for the first requirement the environment fails, and nil when
// it meets them all.
func (r Requirements) Check(env Environment) error {
	if len(r.Targets) > 0 && !slices.Contains(r.Targets, env.Target) {
		return &IncompatibleError{Reason: ReasonTarget, Detail: fmt.Sprintf("runs in %v, not in %q", r.Targets, env.Target)}
	}
	if r.Protocol > ProtocolVersion {
		return &IncompatibleError{Reason: ReasonProtocol, Detail: fmt.Sprintf("uses protocol %d, the application supports up to %d", r.Protocol, ProtocolVersion)}
	}
	return nil
}

// CheckCompatibility returns an *IncompatibleError for the first requirement the environment fails.
func (m Manifest) CheckCompatibility(env Environment) error {
	return m.GetRequirements().Check(env)
}
