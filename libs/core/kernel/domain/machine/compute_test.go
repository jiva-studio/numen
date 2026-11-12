package machine_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
)

type machine struct {
	name string
	hw   machine.Specs
}

func getMachines() []machine {
	return []machine{
		{
			"a sixteen-thread desktop with eight cores and 32 GB",
			machine.Specs{LogicalCPUs: 16, PhysicalCPUs: 8, PerformanceCPUs: 8, TotalRAMMB: 32768, AvailableRAMMB: 24000},
		},
		{
			"an Apple chip with four performance and six efficiency cores",
			machine.Specs{
				LogicalCPUs:     10,
				PhysicalCPUs:    10,
				PerformanceCPUs: 4,
				TotalRAMMB:      16384,
				AvailableRAMMB:  9000,
			},
		},
		{"a two-core laptop", machine.Specs{LogicalCPUs: 2, PhysicalCPUs: 2, PerformanceCPUs: 2, TotalRAMMB: 4096, AvailableRAMMB: 1500}},
		{"a 64-core server", machine.Specs{LogicalCPUs: 128, PhysicalCPUs: 64, PerformanceCPUs: 64, TotalRAMMB: 262144, AvailableRAMMB: 200000}},
		{"a machine whose memory is unknown", machine.Specs{LogicalCPUs: 8, PhysicalCPUs: 4, PerformanceCPUs: 4}},
		{"a machine that reports nothing", machine.Specs{}},
	}
}

func getProfiles() []string {
	return []string{machine.ProfileEco, machine.ProfileBalanced, machine.ProfileMaximum}
}

func TestResolveCompute_NeverExceedsTheCoreBudgetOnAnyMachine(t *testing.T) {
	for _, m := range getMachines() {
		for _, profile := range getProfiles() {
			t.Run(m.name+"/"+profile, func(t *testing.T) {
				res := machine.ResolveCompute(profile, m.hw)

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
	res := machine.ResolveCompute(machine.ProfileMaximum, getMachines()[0].hw)

	if res.CoreBudget != 16 || res.Workers != 16 {
		t.Errorf("budget %d, workers %d; want 16 and 16", res.CoreBudget, res.Workers)
	}
}

func TestResolveCompute_UsesHalfTheDesktopBudgetAtBalanced(t *testing.T) {
	res := machine.ResolveCompute(machine.ProfileBalanced, getMachines()[0].hw)

	if res.CoreBudget != 8 {
		t.Errorf("CoreBudget = %d, want 8", res.CoreBudget)
	}
	if res.Workers != 4 {
		t.Errorf("Workers = %d, want half the budget of 8", res.Workers)
	}
}

func TestResolveCompute_CountsOnlyPerformanceCoresOnAnAppleChip(t *testing.T) {
	hw := getMachines()[1].hw

	maximum := machine.ResolveCompute(machine.ProfileMaximum, hw)
	balanced := machine.ResolveCompute(machine.ProfileBalanced, hw)

	if maximum.CoreBudget != 4 {
		t.Errorf("Maximum budget = %d, want the 4 performance cores", maximum.CoreBudget)
	}
	if balanced.CoreBudget != 2 {
		t.Errorf("Balanced budget = %d, want 2", balanced.CoreBudget)
	}
}

func TestResolveCompute_CountsThreadsOfPerformanceCores(t *testing.T) {
	hw := machine.Specs{LogicalCPUs: 20, PhysicalCPUs: 14, PerformanceCPUs: 6, TotalRAMMB: 32768}

	res := machine.ResolveCompute(machine.ProfileMaximum, hw)

	if res.CoreBudget != 12 {
		t.Errorf("budget = %d, want 6 cores of 2 threads", res.CoreBudget)
	}
}

func TestResolveCompute_RunsEcoSingleThreaded(t *testing.T) {
	for _, m := range getMachines() {
		res := machine.ResolveCompute(machine.ProfileEco, m.hw)

		if res.Workers != 1 || res.CoreBudget != 1 {
			t.Errorf("%s: Eco resolved to %+v, want one of everything", m.name, res)
		}
	}
}

func TestResolveCompute_TreatsAnUnreportedMachineAsOneCPU(t *testing.T) {
	res := machine.ResolveCompute(machine.ProfileMaximum, machine.Specs{})

	if res.CoreBudget != 1 || res.Workers != 1 {
		t.Errorf("resolved %+v, want a budget of one", res)
	}
}

func TestResolveCompute_UnknownProfileFallsBackToBalanced(t *testing.T) {
	for _, profile := range []string{"turbo", "custom"} {
		res := machine.ResolveCompute(profile, getMachines()[0].hw)

		if res.Profile != machine.ProfileBalanced {
			t.Errorf("%q resolved to %q, want balanced", profile, res.Profile)
		}
	}
}

func TestResolveCompute_DerivesMemoryLimitFromRAMAndProfile(t *testing.T) {
	cases := []struct {
		name    string
		profile string
		hw      machine.Specs
		want    int64
	}{
		{"eco on a Ryzen with 32 GB", machine.ProfileEco, machine.Specs{LogicalCPUs: 16, TotalRAMMB: 32768}, 8192},
		{"balanced on a Ryzen with 32 GB", machine.ProfileBalanced, machine.Specs{LogicalCPUs: 16, TotalRAMMB: 32768}, 16384},
		{"maximum on a Ryzen with 32 GB", machine.ProfileMaximum, machine.Specs{LogicalCPUs: 16, TotalRAMMB: 32768}, 24576},
		{"balanced on an M4 with 24 GB", machine.ProfileBalanced, machine.Specs{LogicalCPUs: 10, TotalRAMMB: 24576}, 12288},
		{"maximum on an M4 with 24 GB", machine.ProfileMaximum, machine.Specs{LogicalCPUs: 10, TotalRAMMB: 24576}, 18432},
		{"eco on a laptop with 2 GB", machine.ProfileEco, machine.Specs{LogicalCPUs: 2, TotalRAMMB: 2048}, 512},
		{"balanced on a laptop with 2 GB", machine.ProfileBalanced, machine.Specs{LogicalCPUs: 2, TotalRAMMB: 2048}, 1024},
		{"unknown memory sets no limit at eco", machine.ProfileEco, machine.Specs{LogicalCPUs: 8}, 0},
		{"unknown memory sets no limit at balanced", machine.ProfileBalanced, machine.Specs{LogicalCPUs: 8}, 0},
		{"unknown memory sets no limit at maximum", machine.ProfileMaximum, machine.Specs{LogicalCPUs: 8}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := machine.ResolveCompute(c.profile, c.hw).MemoryLimitMB
			if got != c.want {
				t.Fatalf("MemoryLimitMB = %d, want %d", got, c.want)
			}
		})
	}
}

func TestResolveCompute_RaisesTheMemoryLimitWithTheProfile(t *testing.T) {
	hw := machine.Specs{LogicalCPUs: 8, TotalRAMMB: 16384}
	eco := machine.ResolveCompute(machine.ProfileEco, hw).MemoryLimitMB
	balanced := machine.ResolveCompute(machine.ProfileBalanced, hw).MemoryLimitMB
	maximum := machine.ResolveCompute(machine.ProfileMaximum, hw).MemoryLimitMB
	if !(eco < balanced && balanced < maximum && maximum < hw.TotalRAMMB) {
		t.Fatalf("limits eco=%d balanced=%d maximum=%d are not ordered below %d", eco, balanced, maximum, hw.TotalRAMMB)
	}
}
