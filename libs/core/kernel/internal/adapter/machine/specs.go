// Package machine reads what the machine offers: its CPUs and memory, and the CPU quota of the cgroup it runs in.
package machine

import (
	"runtime"

	machinedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
)

// SpecsReader reports the machine this process runs on.
type SpecsReader struct{}

// ReadSpecs inspects the current host machine capabilities.
func (SpecsReader) ReadSpecs() machinedomain.Specs {
	hw := readPlatformHardware()
	logical := runtime.NumCPU()
	if hw.LogicalCPUs > 0 && hw.LogicalCPUs < logical {
		logical = hw.LogicalCPUs
	}
	hw.LogicalCPUs = logical
	return hw.Normalize()
}
