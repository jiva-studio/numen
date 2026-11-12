package hardware

import (
	"runtime"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// Performance is the settings section for workload and resource profile.
type Performance struct {
	Profile string `json:"profile"`
}

// Reader reports the machine this process runs on.
type Reader struct{}

// ReadMachine inspects the current host machine capabilities.
func (Reader) ReadMachine() settingsdomain.MachineSpecs { return Detect() }

// Detect inspects the current host machine capabilities.
func Detect() settingsdomain.MachineSpecs {
	hw := readPlatformHardware()
	logical := runtime.NumCPU()
	if hw.LogicalCPUs > 0 && hw.LogicalCPUs < logical {
		logical = hw.LogicalCPUs
	}
	hw.LogicalCPUs = logical
	return hw.Normalize()
}
