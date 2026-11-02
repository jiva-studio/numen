package hardware

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

const (
	bytesPerMB = 1024 * 1024
	kBPerMB    = 1024
)

// cpuCore identifies one physical core by the package that holds it and its number within that package.
type cpuCore struct{ Package, Core int }

// parseMeminfo reads MemTotal and MemAvailable from the content of /proc/meminfo, in megabytes. A field that is absent is 0.
func parseMeminfo(content string) (totalMB, availableMB int64) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		name, rest, found := strings.Cut(scanner.Text(), ":")
		if !found {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		kB, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil || kB < 0 {
			continue
		}
		switch name {
		case "MemTotal":
			totalMB = kB / kBPerMB
		case "MemAvailable":
			availableMB = kB / kBPerMB
		}
	}
	return totalMB, availableMB
}

// parseCPUList reads a kernel CPU list such as "0-3,8,10-11" into the CPU numbers it names.
func parseCPUList(content string) ([]int, error) {
	text := strings.TrimSpace(content)
	if text == "" {
		return nil, nil
	}
	var cpus []int
	for _, part := range strings.Split(text, ",") {
		first, last, isRange := strings.Cut(strings.TrimSpace(part), "-")
		from, err := strconv.Atoi(first)
		if err != nil {
			return nil, fmt.Errorf("parse cpu list %q: %w", text, err)
		}
		to := from
		if isRange {
			if to, err = strconv.Atoi(last); err != nil {
				return nil, fmt.Errorf("parse cpu list %q: %w", text, err)
			}
		}
		if to < from {
			return nil, fmt.Errorf("parse cpu list %q: range %d-%d runs backwards", text, from, to)
		}
		for cpu := from; cpu <= to; cpu++ {
			cpus = append(cpus, cpu)
		}
	}
	return cpus, nil
}

// parseTopologyID reads the content of a sysfs topology file, which holds one integer.
func parseTopologyID(content string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(content))
	if err != nil {
		return 0, fmt.Errorf("parse topology id: %w", err)
	}
	return id, nil
}

// countCores is the number of distinct physical cores among the cores of a set of logical CPUs.
func countCores(cores []cpuCore) int {
	observed := make(map[cpuCore]struct{}, len(cores))
	for _, core := range cores {
		observed[core] = struct{}{}
	}
	return len(observed)
}

// parseCPUMax reads the content of a cgroup v2 cpu.max file into the number of CPUs the quota allows, rounded up. A file without a quota gives 0.
func parseCPUMax(content string) int {
	fields := strings.Fields(content)
	if len(fields) != 2 || fields[0] == "max" {
		return 0
	}
	quota, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil || quota <= 0 {
		return 0
	}
	period, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || period <= 0 {
		return 0
	}
	return int((quota + period - 1) / period)
}
