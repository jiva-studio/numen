//go:build windows

package hardware

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

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

// parseWindowsCore parses one core record located at offset in the buffer.
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
