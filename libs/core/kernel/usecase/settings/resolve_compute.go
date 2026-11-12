package settings

import settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"

// MachineReader reports what the machine offers.
type MachineReader interface {
	ReadMachine() settingsdomain.MachineSpecs
}

// ResolveCompute sizes the worker pools for a performance profile on the machine.
type ResolveCompute struct {
	machine MachineReader
}

// NewResolveCompute makes the use case over the source of the machine facts.
func NewResolveCompute(machine MachineReader) *ResolveCompute {
	return &ResolveCompute{machine: machine}
}

// Execute reads the machine and resolves the profile on it. A profile that names none of the presets
// resolves as the balanced one.
func (u *ResolveCompute) Execute(profile string) settingsdomain.ResolvedCompute {
	return settingsdomain.ResolveCompute(profile, u.machine.ReadMachine())
}
