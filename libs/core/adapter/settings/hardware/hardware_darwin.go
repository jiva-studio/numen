//go:build darwin

package hardware

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/unix"
)

// readPlatformHardware reads memory and CPU counts through sysctl. The performance CPUs are those of the highest-ranked performance level; free memory is not reported by sysctl and stays unknown.
func readPlatformHardware() Specs {
	var hw Specs
	if bytes, err := unix.SysctlUint64("hw.memsize"); err == nil {
		hw.TotalRAMMB = int64(bytes / bytesPerMB)
	}
	if physical, err := unix.SysctlUint32("hw.physicalcpu"); err == nil {
		hw.PhysicalCPUs = int(physical)
	}
	hw.PerformanceCPUs = hw.PhysicalCPUs
	if level, ok := pickPerformanceLevel(readPerfLevels()); ok {
		hw.PerformanceCPUs = level.PhysicalCPUs
	}
	if runtime.GOARCH == "arm64" {
		hw.Accelerators = []string{AcceleratorCoreML}
	}
	return hw
}

func readPerfLevels() []perfLevel {
	count, err := unix.SysctlUint32("hw.nperflevels")
	if err != nil {
		return nil
	}
	levels := make([]perfLevel, 0, count)
	for i := 0; i < int(count); i++ {
		prefix := fmt.Sprintf("hw.perflevel%d.", i)
		name, err := unix.Sysctl(prefix + "name")
		if err != nil {
			continue
		}
		physical, err := unix.SysctlUint32(prefix + "physicalcpu")
		if err != nil {
			continue
		}
		logical, err := unix.SysctlUint32(prefix + "logicalcpu")
		if err != nil {
			continue
		}
		levels = append(levels, perfLevel{Name: name, PhysicalCPUs: int(physical), LogicalCPUs: int(logical)})
	}
	return levels
}
