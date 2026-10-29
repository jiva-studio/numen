package hardware_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
)

type machine struct {
	name string
	hw   hardware.Specs
}

var machines = []machine{
	{"a sixteen-thread desktop with eight cores and 32 GB", hardware.Specs{LogicalCPUs: 16, PhysicalCPUs: 8, PerformanceCPUs: 8, TotalRAMMB: 32768, AvailableRAMMB: 24000}},
	{"an Apple chip with four performance and six efficiency cores", hardware.Specs{LogicalCPUs: 10, PhysicalCPUs: 10, PerformanceCPUs: 4, TotalRAMMB: 16384, AvailableRAMMB: 9000, Accelerators: []string{hardware.AcceleratorCoreML}}},
	{"a two-core laptop", hardware.Specs{LogicalCPUs: 2, PhysicalCPUs: 2, PerformanceCPUs: 2, TotalRAMMB: 4096, AvailableRAMMB: 1500}},
	{"a 64-core server", hardware.Specs{LogicalCPUs: 128, PhysicalCPUs: 64, PerformanceCPUs: 64, TotalRAMMB: 262144, AvailableRAMMB: 200000}},
	{"a machine whose memory is unknown", hardware.Specs{LogicalCPUs: 8, PhysicalCPUs: 4, PerformanceCPUs: 4}},
	{"a machine that reports nothing", hardware.Specs{}},
}

var profiles = []hardware.PerformanceProfile{hardware.ProfileEco, hardware.ProfileBalanced, hardware.ProfileMaximum}

func TestResolveCompute_NeverExceedsTheCoreBudgetOnAnyMachine(t *testing.T) {
	for _, m := range machines {
		for _, profile := range profiles {
			t.Run(m.name+"/"+profile, func(t *testing.T) {
				res := hardware.ResolveCompute(profile, m.hw)

				if res.Profile != profile {
					t.Errorf("profile = %q, want %q", res.Profile, profile)
				}
				if res.CoreBudget < 1 || res.Workers < 1 {
					t.Errorf("budget %d, workers %d; want at least 1 each", res.CoreBudget, res.Workers)
				}
				if res.Workers > res.CoreBudget {
					t.Errorf("Workers = %d, over the budget of %d", res.Workers, res.CoreBudget)
				}
			})
		}
	}
}

func TestResolveComputeSizesWorkersToTheFullBudgetAtMaximum(t *testing.T) {
	res := hardware.ResolveCompute(hardware.ProfileMaximum, machines[0].hw)

	if res.CoreBudget != 16 || res.Workers != 16 {
		t.Errorf("budget %d, workers %d; want 16 and 16", res.CoreBudget, res.Workers)
	}
}

func TestResolveCompute_UsesHalfTheDesktopBudgetAtBalanced(t *testing.T) {
	res := hardware.ResolveCompute(hardware.ProfileBalanced, machines[0].hw)

	if res.CoreBudget != 8 {
		t.Errorf("CoreBudget = %d, want 8", res.CoreBudget)
	}
	if res.Workers != 4 {
		t.Errorf("Workers = %d, want half the budget of 8", res.Workers)
	}
}

func TestResolveCompute_CountsOnlyPerformanceCoresOnAnAppleChip(t *testing.T) {
	hw := machines[1].hw

	maximum := hardware.ResolveCompute(hardware.ProfileMaximum, hw)
	balanced := hardware.ResolveCompute(hardware.ProfileBalanced, hw)

	if maximum.CoreBudget != 4 {
		t.Errorf("Maximum budget = %d, want the 4 performance cores", maximum.CoreBudget)
	}
	if balanced.CoreBudget != 2 {
		t.Errorf("Balanced budget = %d, want 2", balanced.CoreBudget)
	}
}

func TestResolveCompute_CountsThreadsOfPerformanceCores(t *testing.T) {
	hw := hardware.Specs{LogicalCPUs: 20, PhysicalCPUs: 14, PerformanceCPUs: 6, TotalRAMMB: 32768}

	res := hardware.ResolveCompute(hardware.ProfileMaximum, hw)

	if res.CoreBudget != 12 {
		t.Errorf("budget = %d, want 6 cores of 2 threads", res.CoreBudget)
	}
}

func TestResolveCompute_RunsEcoSingleThreaded(t *testing.T) {
	for _, m := range machines {
		res := hardware.ResolveCompute(hardware.ProfileEco, m.hw)

		if res.Workers != 1 || res.CoreBudget != 1 {
			t.Errorf("%s: Eco resolved to %+v, want one of everything", m.name, res)
		}
	}
}

func TestResolveCompute_TreatsAnUnreportedMachineAsOneCPU(t *testing.T) {
	res := hardware.ResolveCompute(hardware.ProfileMaximum, hardware.Specs{})

	if res.CoreBudget != 1 || res.Workers != 1 {
		t.Errorf("resolved %+v, want a budget of one", res)
	}
}

func TestResolveCompute_UnknownProfileFallsBackToBalanced(t *testing.T) {
	for _, profile := range []hardware.PerformanceProfile{"turbo", "custom"} {
		res := hardware.ResolveCompute(profile, machines[0].hw)

		if res.Profile != hardware.ProfileBalanced {
			t.Errorf("%q resolved to %q, want balanced", profile, res.Profile)
		}
	}
}

func TestDetectHardware_ReportsConsistentCounts(t *testing.T) {
	hw := hardware.Detect()

	if hw.LogicalCPUs < 1 || hw.PhysicalCPUs < 1 || hw.PerformanceCPUs < 1 {
		t.Fatalf("detected %+v, want every count at least 1", hw)
	}
	if hw.PhysicalCPUs > hw.LogicalCPUs || hw.PerformanceCPUs > hw.PhysicalCPUs {
		t.Errorf("detected %+v, want performance <= physical <= logical", hw)
	}
	if hw.TotalRAMMB > 0 && hw.AvailableRAMMB > hw.TotalRAMMB {
		t.Errorf("detected %+v, want available memory within installed memory", hw)
	}
}

func TestResolveCompute_DerivesMemoryLimitFromRAMAndProfile(t *testing.T) {
	cases := []struct {
		name    string
		profile hardware.PerformanceProfile
		hw      hardware.Specs
		want    int64
	}{
		{"eco on a Ryzen with 32 GB", hardware.ProfileEco, hardware.Specs{LogicalCPUs: 16, TotalRAMMB: 32768}, 8192},
		{"balanced on a Ryzen with 32 GB", hardware.ProfileBalanced, hardware.Specs{LogicalCPUs: 16, TotalRAMMB: 32768}, 16384},
		{"maximum on a Ryzen with 32 GB", hardware.ProfileMaximum, hardware.Specs{LogicalCPUs: 16, TotalRAMMB: 32768}, 24576},
		{"balanced on an M4 with 24 GB", hardware.ProfileBalanced, hardware.Specs{LogicalCPUs: 10, TotalRAMMB: 24576}, 12288},
		{"maximum on an M4 with 24 GB", hardware.ProfileMaximum, hardware.Specs{LogicalCPUs: 10, TotalRAMMB: 24576}, 18432},
		{"eco on a laptop with 2 GB", hardware.ProfileEco, hardware.Specs{LogicalCPUs: 2, TotalRAMMB: 2048}, 512},
		{"balanced on a laptop with 2 GB", hardware.ProfileBalanced, hardware.Specs{LogicalCPUs: 2, TotalRAMMB: 2048}, 1024},
		{"unknown memory sets no limit at eco", hardware.ProfileEco, hardware.Specs{LogicalCPUs: 8}, 0},
		{"unknown memory sets no limit at balanced", hardware.ProfileBalanced, hardware.Specs{LogicalCPUs: 8}, 0},
		{"unknown memory sets no limit at maximum", hardware.ProfileMaximum, hardware.Specs{LogicalCPUs: 8}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := hardware.ResolveCompute(c.profile, c.hw).MemoryLimitMB
			if got != c.want {
				t.Fatalf("MemoryLimitMB = %d, want %d", got, c.want)
			}
		})
	}
}

func TestResolveCompute_RaisesTheMemoryLimitWithTheProfile(t *testing.T) {
	hw := hardware.Specs{LogicalCPUs: 8, TotalRAMMB: 16384}
	eco := hardware.ResolveCompute(hardware.ProfileEco, hw).MemoryLimitMB
	balanced := hardware.ResolveCompute(hardware.ProfileBalanced, hw).MemoryLimitMB
	maximum := hardware.ResolveCompute(hardware.ProfileMaximum, hw).MemoryLimitMB
	if !(eco < balanced && balanced < maximum && maximum < hw.TotalRAMMB) {
		t.Fatalf("limits eco=%d balanced=%d maximum=%d are not ordered below %d", eco, balanced, maximum, hw.TotalRAMMB)
	}
}
