package hardware

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math/bits"
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

// perfLevel is one sysctl performance level of an Apple chip.
type perfLevel struct {
	Name         string
	PhysicalCPUs int
	LogicalCPUs  int
}

// getPerfLevelRank orders the level names Apple chips use; a name outside the list ranks lowest.
func getPerfLevelRank(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "super":
		return 3
	case "performance":
		return 2
	case "efficiency":
		return 1
	default:
		return 0
	}
}

// pickPerformanceLevel selects the level with the highest-ranked name, and the lowest index among equal ranks.
func pickPerformanceLevel(levels []perfLevel) (perfLevel, bool) {
	best := -1
	for i, level := range levels {
		if level.PhysicalCPUs < 1 {
			continue
		}
		if best < 0 || getPerfLevelRank(level.Name) > getPerfLevelRank(levels[best].Name) {
			best = i
		}
	}
	if best < 0 {
		return perfLevel{}, false
	}
	return levels[best], true
}

// windowsCore is one physical core reported by GetLogicalProcessorInformationEx.
type windowsCore struct {
	// EfficiencyClass is higher for cores that favour performance.
	EfficiencyClass int
	// LogicalCPUs is the number of hardware threads of the core.
	LogicalCPUs int
}

const (
	windowsRelationProcessorCore = 0
	windowsRecordHeaderSize      = 8
	// The offsets below lie within the record that follows the header.
	windowsEfficiencyClassOffset = 1
	windowsGroupCountOffset      = 22
	windowsGroupMaskOffset       = 24
)

// parseWindowsCores reads the buffer GetLogicalProcessorInformationEx fills for RelationProcessorCore. maskSize is the size of a pointer on the host.
func parseWindowsCores(buffer []byte, maskSize int) ([]windowsCore, error) {
	var cores []windowsCore
	for offset := 0; offset < len(buffer); {
		if len(buffer)-offset < windowsRecordHeaderSize {
			return nil, fmt.Errorf("parse processor information: truncated header at %d", offset)
		}
		relationship := binary.LittleEndian.Uint32(buffer[offset:])
		size := int(binary.LittleEndian.Uint32(buffer[offset+4:]))
		if size < windowsRecordHeaderSize || offset+size > len(buffer) {
			return nil, fmt.Errorf("parse processor information: record of %d bytes at %d", size, offset)
		}
		if relationship == windowsRelationProcessorCore {
			core, err := parseWindowsCore(buffer[offset+windowsRecordHeaderSize:offset+size], offset, maskSize)
			if err != nil {
				return nil, err
			}
			cores = append(cores, core)
		}
		offset += size
	}
	return cores, nil
}

// parseWindowsCore reads one core record, which lies at offset in the buffer.
func parseWindowsCore(record []byte, offset, maskSize int) (windowsCore, error) {
	if len(record) < windowsGroupMaskOffset {
		return windowsCore{}, fmt.Errorf("parse processor information: short core record at %d", offset)
	}
	groupAffinitySize := maskSize + 8
	core := windowsCore{EfficiencyClass: int(record[windowsEfficiencyClassOffset])}
	groups := int(binary.LittleEndian.Uint16(record[windowsGroupCountOffset:]))
	for g := 0; g < groups; g++ {
		start := windowsGroupMaskOffset + g*groupAffinitySize
		if start+maskSize > len(record) {
			return windowsCore{}, fmt.Errorf("parse processor information: short group mask at %d", offset)
		}
		core.LogicalCPUs += bits.OnesCount64(readWindowsMask(record[start:], maskSize))
	}
	return core, nil
}

// readWindowsMask is the affinity mask at the start of a group, as wide as a
// pointer on the host.
func readWindowsMask(group []byte, maskSize int) uint64 {
	if maskSize == 8 {
		return binary.LittleEndian.Uint64(group)
	}
	return uint64(binary.LittleEndian.Uint32(group))
}

// countPerformanceCores is the number of cores whose efficiency class is the highest present.
func countPerformanceCores(cores []windowsCore) int {
	highest := -1
	count := 0
	for _, core := range cores {
		switch {
		case core.EfficiencyClass > highest:
			highest, count = core.EfficiencyClass, 1
		case core.EfficiencyClass == highest:
			count++
		}
	}
	return count
}
