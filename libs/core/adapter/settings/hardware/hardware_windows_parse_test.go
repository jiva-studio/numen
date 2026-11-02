//go:build windows

package hardware

import (
	"encoding/binary"
	"slices"
	"testing"
)

// windowsRecord builds a RelationProcessorCore record with one group mask.
func windowsRecord(efficiencyClass byte, mask uint64, maskSize int) []byte {
	size := 32 + maskSize
	record := make([]byte, size)
	binary.LittleEndian.PutUint32(record[0:], 0)
	binary.LittleEndian.PutUint32(record[4:], uint32(size))
	record[8+1] = efficiencyClass
	binary.LittleEndian.PutUint16(record[8+22:], 1)
	if maskSize == 8 {
		binary.LittleEndian.PutUint64(record[8+24:], mask)
	} else {
		binary.LittleEndian.PutUint32(record[8+24:], uint32(mask))
	}
	return record
}

func TestParseWindowsCores_ReadsEfficiencyClassAndThreads(t *testing.T) {
	var buffer []byte
	buffer = append(buffer, windowsRecord(1, 0b11, 8)...)
	buffer = append(buffer, windowsRecord(1, 0b1100, 8)...)
	buffer = append(buffer, windowsRecord(0, 0b10000, 8)...)

	cores, err := parseWindowsCores(buffer, 8)

	if err != nil {
		t.Fatal(err)
	}
	want := []windowsCore{{1, 2}, {1, 2}, {0, 1}}
	if !slices.Equal(cores, want) {
		t.Errorf("got %v, want %v", cores, want)
	}
	if got := countPerformanceCores(cores); got != 2 {
		t.Errorf("countPerformanceCores = %d, want 2", got)
	}
}

func TestParseWindowsCores_ReadsMasksOfAThirtyTwoBitHost(t *testing.T) {
	cores, err := parseWindowsCores(windowsRecord(0, 0b111, 4), 4)

	if err != nil || len(cores) != 1 || cores[0].LogicalCPUs != 3 {
		t.Errorf("got %v, %v; want one core of 3 threads", cores, err)
	}
}

func TestParseWindowsCores_SkipsOtherRelationships(t *testing.T) {
	other := windowsRecord(0, 1, 8)
	binary.LittleEndian.PutUint32(other[0:], 2)

	cores, err := parseWindowsCores(append(other, windowsRecord(0, 1, 8)...), 8)

	if err != nil || len(cores) != 1 {
		t.Errorf("got %v, %v; want one core", cores, err)
	}
}

func TestParseWindowsCores_RejectsTruncatedBuffers(t *testing.T) {
	record := windowsRecord(0, 1, 8)
	for _, buffer := range [][]byte{record[:5], record[:len(record)-4]} {
		if _, err := parseWindowsCores(buffer, 8); err == nil {
			t.Errorf("parseWindowsCores accepted %d truncated bytes", len(buffer))
		}
	}
}

func TestCountPerformanceCores_IsZeroWithoutCores(t *testing.T) {
	if got := countPerformanceCores(nil); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}
