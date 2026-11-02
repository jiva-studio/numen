package hardware

import (
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
