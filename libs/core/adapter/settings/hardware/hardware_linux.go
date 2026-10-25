//go:build linux

package hardware

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/sys/unix"
)

const (
	linuxMeminfoPath    = "/proc/meminfo"
	linuxCPUDir         = "/sys/devices/system/cpu"
	linuxPerformanceCPU = "/sys/devices/cpu_core/cpus"
	linuxCPUMaxPath     = "/sys/fs/cgroup/cpu.max"
)

// readPlatformHardware reads memory and CPU topology from procfs and sysfs. The CPUs counted are those the process may run on, limited by the cgroup v2 CPU quota of the root of the cgroup mount; cgroup v1 and memory limits are not read.
func readPlatformHardware() Specs {
	var hw Specs
	if content, err := os.ReadFile(linuxMeminfoPath); err == nil {
		hw.TotalRAMMB, hw.AvailableRAMMB = parseMeminfo(string(content))
	}

	cores := readLinuxCores()
	hw.PhysicalCPUs = countCores(cores.all)
	hw.PerformanceCPUs = countCores(cores.performance)
	if len(cores.performance) == 0 {
		hw.PerformanceCPUs = hw.PhysicalCPUs
	}

	if content, err := os.ReadFile(linuxCPUMaxPath); err == nil {
		if limit := parseCPUMax(string(content)); limit > 0 {
			hw.LogicalCPUs = limit
		}
	}
	return hw
}

type linuxCores struct {
	all         []cpuCore
	performance []cpuCore
}

// readLinuxCores lists the core of every CPU in the process affinity, and separately those of the performance CPUs of a hybrid chip.
func readLinuxCores() linuxCores {
	allowed := readAffinity()
	performance := readPerformanceCPUs()

	var found linuxCores
	dirs, _ := filepath.Glob(filepath.Join(linuxCPUDir, "cpu[0-9]*"))
	for _, dir := range dirs {
		id, err := strconv.Atoi(filepath.Base(dir)[len("cpu"):])
		if err != nil || (allowed != nil && !allowed[id]) {
			continue
		}
		core, err := readCPUCore(dir)
		if err != nil {
			continue
		}
		found.all = append(found.all, core)
		if performance[id] {
			found.performance = append(found.performance, core)
		}
	}
	return found
}

func readCPUCore(dir string) (cpuCore, error) {
	var core cpuCore
	for file, target := range map[string]*int{"physical_package_id": &core.Package, "core_id": &core.Core} {
		content, err := os.ReadFile(filepath.Join(dir, "topology", file))
		if err != nil {
			return cpuCore{}, fmt.Errorf("read %s: %w", file, err)
		}
		if *target, err = parseTopologyID(string(content)); err != nil {
			return cpuCore{}, err
		}
	}
	return core, nil
}

// readAffinity is the set of CPUs the process may run on, or nil when it cannot be read.
func readAffinity() map[int]bool {
	var set unix.CPUSet
	if err := unix.SchedGetaffinity(0, &set); err != nil {
		return nil
	}
	allowed := make(map[int]bool)
	for cpu := 0; cpu < len(set)*64; cpu++ {
		if set.IsSet(cpu) {
			allowed[cpu] = true
		}
	}
	return allowed
}

// readPerformanceCPUs is the set of CPUs the kernel lists as performance CPUs of a hybrid chip; it is empty on other chips.
func readPerformanceCPUs() map[int]bool {
	content, err := os.ReadFile(linuxPerformanceCPU)
	if err != nil {
		return nil
	}
	cpus, err := parseCPUList(string(content))
	if err != nil {
		return nil
	}
	set := make(map[int]bool, len(cpus))
	for _, cpu := range cpus {
		set[cpu] = true
	}
	return set
}
