package plugin

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// RequirementSource reports which conditions of the host that a manifest names hold.
type RequirementSource interface {
	// IsRequirementMet reports whether the condition holds now.
	IsRequirementMet(ctx context.Context, requirement string) bool
}

// Requirements returns the first condition a backend requires that does not hold. Until a source is set none holds.
type Requirements struct {
	mu     sync.RWMutex
	source RequirementSource
}

// NewRequirements creates the requirements with no source.
func NewRequirements() *Requirements { return &Requirements{} }

// SetSource sets the source that reports the conditions. It replaces the source set before.
func (r *Requirements) SetSource(source RequirementSource) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.source = source
}

// GetUnmet returns the first condition the backend requires that does not hold, and "" where all hold or it requires none.
func (r *Requirements) GetUnmet(ctx context.Context, backend manifest.Backend) string {
	if len(backend.Requires) == 0 {
		return ""
	}
	if r == nil {
		return backend.Requires[0]
	}
	r.mu.RLock()
	source := r.source
	r.mu.RUnlock()
	for _, requirement := range backend.Requires {
		if source == nil || !source.IsRequirementMet(ctx, requirement) {
			return requirement
		}
	}
	return ""
}

// Presence reports whether this device has an identity or is on the way to one.
type Presence interface {
	IsPresent(ctx context.Context) bool
}

// IdentityRequirement meets the identity requirement while the device is present.
type IdentityRequirement struct{ Presence Presence }

// IsRequirementMet reports whether the requirement is the identity and the device is present.
func (r IdentityRequirement) IsRequirementMet(ctx context.Context, requirement string) bool {
	return requirement == manifest.RequirementIdentity && r.Presence.IsPresent(ctx)
}
