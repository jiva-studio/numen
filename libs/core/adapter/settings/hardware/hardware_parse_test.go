package hardware

import (
	"encoding/binary"
	"slices"
	"testing"
)

const meminfoSample = `MemTotal:       32768000 kB
MemFree:         1048576 kB
MemAvailable:   24576000 kB
Buffers:          204800 kB
SwapTotal:       8388604 kB
`

func TestParseMeminfo_ReadsTotalAndAvailableInMegabytes(t *testing.T) {
	total, available := parseMeminfo(meminfoSample)

	if total != 32000 || available != 24000 {
		t.Errorf("total %d, available %d; want 32000 and 24000", total, available)
	}
}

func TestParseMeminfo_PreservesAMissingFieldUnknown(t *testing.T) {
	total, available := parseMeminfo("MemTotal:  2048000 kB\nMemFree: 1024 kB\n")

	if total != 2000 || available != 0 {
		t.Errorf("total %d, available %d; want 2000 and 0", total, available)
	}
}

func TestParseMeminfo_IgnoresGarbage(t *testing.T) {
	total, available := parseMeminfo("not a meminfo\nMemTotal: lots kB\n\n:\n")

	if total != 0 || available != 0 {
		t.Errorf("total %d, available %d; want 0 and 0", total, available)
	}
}

func TestParseCPUList_ExpandsRangesAndSingles(t *testing.T) {
	got, err := parseCPUList("0-3,8,10-11\n")

	if err != nil {
		t.Fatal(err)
	}
	if want := []int{0, 1, 2, 3, 8, 10, 11}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseCPUList_RejectsMalformedLists(t *testing.T) {
	for _, content := range []string{"a-b", "5-2", "1,,2", "0-"} {
		if _, err := parseCPUList(content); err == nil {
			t.Errorf("parseCPUList(%q) succeeded, want an error", content)
		}
	}
}

func TestParseCPUList_ReadsAnEmptyFileAsNoCPUs(t *testing.T) {
	got, err := parseCPUList("\n")

	if err != nil || len(got) != 0 {
		t.Errorf("got %v, %v; want no CPUs and no error", got, err)
	}
}

func TestParseTopologyID_ReadsOneInteger(t *testing.T) {
	got, err := parseTopologyID("7\n")

	if err != nil || got != 7 {
		t.Errorf("got %d, %v; want 7", got, err)
	}
	if _, err := parseTopologyID("seven"); err == nil {
		t.Error("parseTopologyID accepted a word")
	}
}

func TestCountCores_JoinsThreadsOfOneCoreAndSeparatesPackages(t *testing.T) {
	// Eight cores with two threads each on package 0, and a second package that reuses core number 0.
	var cpus []cpuCore
	for thread := 0; thread < 2; thread++ {
		for core := 0; core < 8; core++ {
			cpus = append(cpus, cpuCore{Package: 0, Core: core})
		}
	}
	cpus = append(cpus, cpuCore{Package: 1, Core: 0})

	if got := countCores(cpus); got != 9 {
		t.Errorf("countCores = %d, want 9", got)
	}
}

func TestParseCPUMax_RoundsTheQuotaUpToWholeCPUs(t *testing.T) {
	tests := map[string]int{
		"max 100000\n":   0,
		"200000 100000":  2,
		"150000 100000":  2,
		"50000 100000":   1,
		"":               0,
		"0 100000":       0,
		"100000 0":       0,
		"fast 100000":    0,
		"100000 100000 ": 1,
	}
	for content, want := range tests {
		if got := parseCPUMax(content); got != want {
			t.Errorf("parseCPUMax(%q) = %d, want %d", content, got, want)
		}
	}
}

func TestPickPerformanceLevel_ChoosesTheHighestRankedName(t *testing.T) {
	tests := []struct {
		name   string
		levels []perfLevel
		want   string
		cores  int
	}{
		{"an older chip names its levels Performance and Efficiency", []perfLevel{{"Performance", 4, 4}, {"Efficiency", 6, 6}}, "Performance", 4},
		{"a newer chip puts Super ahead of Performance", []perfLevel{{"Super", 6, 6}, {"Performance", 12, 12}, {"Efficiency", 4, 4}}, "Super", 6},
		{"the order of the levels does not decide", []perfLevel{{"Efficiency", 6, 6}, {"Performance", 4, 4}}, "Performance", 4},
		{"unnamed levels fall to the first", []perfLevel{{"", 8, 8}, {"", 4, 4}}, "", 8},
		{"names are compared without case or padding", []perfLevel{{"efficiency\n", 6, 6}, {" PERFORMANCE", 4, 4}}, " PERFORMANCE", 4},
		{"a level without cores is skipped", []perfLevel{{"Super", 0, 0}, {"Performance", 4, 4}}, "Performance", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pickPerformanceLevel(tt.levels)
			if !ok || got.Name != tt.want || got.PhysicalCPUs != tt.cores {
				t.Errorf("got %+v, %v; want %q with %d cores", got, ok, tt.want, tt.cores)
			}
		})
	}
}

func TestPickPerformanceLevel_ReportsNothingWithoutLevels(t *testing.T) {
	if _, ok := pickPerformanceLevel(nil); ok {
		t.Error("pickPerformanceLevel found a level in an empty list")
	}
}

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
