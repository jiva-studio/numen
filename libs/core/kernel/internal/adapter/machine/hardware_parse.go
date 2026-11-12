package machine

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

func parseCPUList(content string) ([]int, error) {
	text := strings.TrimSpace(content)
	if text == "" {
		return nil, nil
	}
	var cpus []int
	for part := range strings.SplitSeq(text, ",") {
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

func parseTopologyID(content string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(content))
	if err != nil {
		return 0, fmt.Errorf("parse topology id: %w", err)
	}
	return id, nil
}

func countCores(cores []cpuCore) int {
	observed := make(map[cpuCore]struct{}, len(cores))
	for _, core := range cores {
		observed[core] = struct{}{}
	}
	return len(observed)
}

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
