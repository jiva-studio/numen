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
	// ReasonProtocol is a plugin that speaks a newer plugin protocol than the application.
	ReasonProtocol IncompatibleReason = "protocol"
	// ReasonAppTooOld is an application whose version is below the minimum the plugin runs in.
	ReasonAppTooOld IncompatibleReason = "app-too-old"
	// ReasonAppTooNew is an application whose version is above the maximum the plugin runs in.
	ReasonAppTooNew IncompatibleReason = "app-too-new"
)

// ErrIncompatible is a plugin that an application cannot run.
var ErrIncompatible = errors.New("plugin is incompatible with the application")

// IncompatibleError is a plugin that an application cannot run, and why.
type IncompatibleError struct {
	Reason IncompatibleReason
	// Detail states the requirement the application fails.
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
	// Target is the window the plugin is loaded into.
	Target string
	// AppVersion is the version of the application. Empty is an application that states none, and no
	// bound on the versions applies.
	AppVersion string
}

// Requirements are what a plugin requires of the application that runs it.
type Requirements struct {
	// Targets are the windows the plugin runs in. Empty is every window.
	Targets []string
	// Protocol is the plugin protocol the plugin speaks. Zero is the first.
	Protocol int
	// AppVersion bounds the versions of the application.
	AppVersion AppVersionRange
}

// GetRequirements returns what the plugin requires of the application.
func (m Manifest) GetRequirements() Requirements {
	return Requirements{Targets: slices.Clone(m.Targets), Protocol: m.GetProtocol(), AppVersion: m.AppVersion}
}

// Check returns an *IncompatibleError for the first requirement the environment fails, and nil when
// it meets them all.
func (r Requirements) Check(env Environment) error {
	if len(r.Targets) > 0 && !slices.Contains(r.Targets, env.Target) {
		return &IncompatibleError{Reason: ReasonTarget, Detail: fmt.Sprintf("runs in %v, not in %q", r.Targets, env.Target)}
	}
	if r.Protocol > ProtocolVersion {
		return &IncompatibleError{Reason: ReasonProtocol, Detail: fmt.Sprintf("speaks protocol %d, the application speaks up to %d", r.Protocol, ProtocolVersion)}
	}
	if env.AppVersion == "" {
		return nil
	}
	switch r.AppVersion.Locate(env.AppVersion) {
	case -1:
		return &IncompatibleError{Reason: ReasonAppTooOld, Detail: fmt.Sprintf("needs application %s or newer, this is %s", r.AppVersion.Min, env.AppVersion)}
	case 1:
		return &IncompatibleError{Reason: ReasonAppTooNew, Detail: fmt.Sprintf("needs application %s or older, this is %s", r.AppVersion.Max, env.AppVersion)}
	}
	return nil
}

// CheckCompatibility returns an *IncompatibleError for the first requirement the environment fails.
func (m Manifest) CheckCompatibility(env Environment) error {
	return m.GetRequirements().Check(env)
}
