//go:build linux

package hardware

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

const (
	linuxMeminfoPath    = "/proc/meminfo"
	linuxCPUDir         = "/sys/devices/system/cpu"
	linuxPerformanceCPU = "/sys/devices/cpu_core/cpus"
	linuxCPUMaxPath     = "/sys/fs/cgroup/cpu.max"
)

func readPlatformHardware() settingsdomain.MachineSpecs {
	var hw settingsdomain.MachineSpecs
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

func readLinuxCores() linuxCores {
	allowed := readAffinity()
	performance := readPerformanceCPUs()

	var found linuxCores
	entries, err := os.ReadDir(linuxCPUDir)
	if err != nil {
		return found
	}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) <= len("cpu") || !strings.HasPrefix(name, "cpu") || name[len("cpu")] < '0' || name[len("cpu")] > '9' {
			continue
		}
		id, err := strconv.Atoi(name[len("cpu"):])
		if err != nil || (allowed != nil && !allowed[id]) {
			continue
		}
		core, err := readCPUCore(filepath.Join(linuxCPUDir, name))
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

func readAffinity() map[int]bool {
	var set unix.CPUSet
	if err := unix.SchedGetaffinity(0, &set); err != nil {
		return nil
	}
	allowed := make(map[int]bool)
	for cpu := range len(set) * 64 {
		if set.IsSet(cpu) {
			allowed[cpu] = true
		}
	}
	return allowed
}

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
