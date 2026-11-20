package settings

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
)

// ResolveCompute sizes the worker pools for a performance profile on the machine.
type ResolveCompute struct {
	machine SpecsReader
}

// NewResolveCompute makes the use case over the source of the machine facts.
func NewResolveCompute(specs SpecsReader) *ResolveCompute {
	return &ResolveCompute{machine: specs}
}

// Execute reads the machine and resolves the profile on it. A profile that names none of the presets
// resolves as the balanced one.
func (u *ResolveCompute) Execute(profile string) machine.ResolvedCompute {
	return machine.ResolveCompute(profile, u.machine.ReadSpecs())
}
