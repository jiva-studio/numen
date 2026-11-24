package plugin

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// RequirementSource says whether a condition of the host that a manifest names holds.
type RequirementSource interface {
	// IsRequirementMet reports whether the condition holds now.
	IsRequirementMet(ctx context.Context, requirement string) bool
}

// Requirements answers whether the conditions a backend requires hold. Until a source is set none holds.
type Requirements struct {
	mu     sync.RWMutex
	source RequirementSource
}

// NewRequirements creates the requirements with no source.
func NewRequirements() *Requirements { return &Requirements{} }

// SetSource sets what says whether a condition holds. It replaces the source set before.
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
