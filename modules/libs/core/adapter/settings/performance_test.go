package settings_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
)

type machine struct {
	name string
	hw   settings.HardwareSpecs
}

var machines = []machine{
	{"a sixteen-thread desktop with eight cores and 32 GB", settings.HardwareSpecs{LogicalCPUs: 16, PhysicalCPUs: 8, PerformanceCPUs: 8, TotalRAMMB: 32768, AvailableRAMMB: 24000}},
	{"an Apple chip with four performance and six efficiency cores", settings.HardwareSpecs{LogicalCPUs: 10, PhysicalCPUs: 10, PerformanceCPUs: 4, TotalRAMMB: 16384, AvailableRAMMB: 9000, Accelerators: []string{settings.AcceleratorCoreML}}},
	{"a two-core laptop", settings.HardwareSpecs{LogicalCPUs: 2, PhysicalCPUs: 2, PerformanceCPUs: 2, TotalRAMMB: 4096, AvailableRAMMB: 1500}},
	{"a 64-core server", settings.HardwareSpecs{LogicalCPUs: 128, PhysicalCPUs: 64, PerformanceCPUs: 64, TotalRAMMB: 262144, AvailableRAMMB: 200000}},
	{"a machine whose memory is unknown", settings.HardwareSpecs{LogicalCPUs: 8, PhysicalCPUs: 4, PerformanceCPUs: 4}},
	{"a machine that reports nothing", settings.HardwareSpecs{}},
}

var profiles = []settings.PerformanceProfile{settings.ProfileEco, settings.ProfileBalanced, settings.ProfileMaximum, settings.ProfileCustom}

func TestResolveCompute_NeverExceedsTheCoreBudgetOnAnyMachine(t *testing.T) {
	for _, m := range machines {
		for _, profile := range profiles {
			t.Run(m.name+"/"+profile, func(t *testing.T) {
				res := settings.ResolveCompute(profile, m.hw, settings.PerformanceCustom{})

				if res.Profile != profile {
					t.Errorf("profile = %q, want %q", res.Profile, profile)
				}
				for name, value := range map[string]int{
					"CoreBudget": res.CoreBudget, "EmbeddingBatchSize": res.EmbeddingBatchSize,
					"OcrThreads": res.OcrThreads, "OcrSessions": res.OcrSessions,
					"LlmConcurrency": res.LlmConcurrency, "Workers": res.Workers,
				} {
					if value < 1 {
						t.Errorf("%s = %d, want at least 1", name, value)
					}
				}
				if got := res.OcrThreads * res.OcrSessions; got > res.CoreBudget {
					t.Errorf("OCR threads x sessions = %d, over the budget of %d", got, res.CoreBudget)
				}
				if res.Workers > res.CoreBudget {
					t.Errorf("Workers = %d, over the budget of %d", res.Workers, res.CoreBudget)
				}
			})
		}
	}
}

func TestResolveCompute_FillsTheDesktopBudgetWithFourSessionsOfFourThreadsAtMaximum(t *testing.T) {
	res := settings.ResolveCompute(settings.ProfileMaximum, machines[0].hw, settings.PerformanceCustom{})

	if res.CoreBudget != 16 || res.OcrSessions != 4 || res.OcrThreads != 4 {
		t.Errorf("budget %d, sessions %d, threads %d; want 16, 4, 4", res.CoreBudget, res.OcrSessions, res.OcrThreads)
	}
	if res.Workers != 16 {
		t.Errorf("Workers = %d, want the whole budget of 16", res.Workers)
	}
}

func TestResolveCompute_UsesHalfTheDesktopBudgetAtBalanced(t *testing.T) {
	res := settings.ResolveCompute(settings.ProfileBalanced, machines[0].hw, settings.PerformanceCustom{})

	if got := res.OcrThreads * res.OcrSessions; got != 8 {
		t.Errorf("OCR threads x sessions = %d, want 8", got)
	}
	if res.Workers != 4 {
		t.Errorf("Workers = %d, want half the budget of 8", res.Workers)
	}
}

func TestResolveCompute_CountsOnlyPerformanceCoresOnAnAppleChip(t *testing.T) {
	hw := machines[1].hw

	maximum := settings.ResolveCompute(settings.ProfileMaximum, hw, settings.PerformanceCustom{})
	balanced := settings.ResolveCompute(settings.ProfileBalanced, hw, settings.PerformanceCustom{})

	if maximum.CoreBudget != 4 {
		t.Errorf("Maximum budget = %d, want the 4 performance cores", maximum.CoreBudget)
	}
	if balanced.CoreBudget != 2 {
		t.Errorf("Balanced budget = %d, want 2", balanced.CoreBudget)
	}
}

func TestResolveCompute_CountsHardwareThreadsOfPerformanceCoresOnAHybridChip(t *testing.T) {
	hw := settings.HardwareSpecs{LogicalCPUs: 20, PhysicalCPUs: 14, PerformanceCPUs: 6, TotalRAMMB: 32768}

	res := settings.ResolveCompute(settings.ProfileMaximum, hw, settings.PerformanceCustom{})

	if res.CoreBudget != 12 {
		t.Errorf("budget = %d, want 6 cores of 2 threads", res.CoreBudget)
	}
}

func TestResolveCompute_RunsEcoSingleThreaded(t *testing.T) {
	for _, m := range machines {
		res := settings.ResolveCompute(settings.ProfileEco, m.hw, settings.PerformanceCustom{})

		if res.OcrThreads != 1 || res.OcrSessions != 1 || res.LlmConcurrency != 1 || res.Workers != 1 || res.CoreBudget != 1 {
			t.Errorf("%s: Eco resolved to %+v, want one of everything", m.name, res)
		}
		if res.EmbeddingBatchSize != 16 {
			t.Errorf("%s: Eco batch = %d, want 16", m.name, res.EmbeddingBatchSize)
		}
	}
}

func TestResolveCompute_SplitsALargeBudgetIntoMoreSessionsOfBoundedThreads(t *testing.T) {
	res := settings.ResolveCompute(settings.ProfileMaximum, machines[3].hw, settings.PerformanceCustom{})

	if res.OcrThreads != 8 || res.OcrSessions != 16 {
		t.Errorf("threads %d, sessions %d; want 8 and 16", res.OcrThreads, res.OcrSessions)
	}
}

func TestResolveCompute_ScalesTheBatchWithFreeMemoryWithinBounds(t *testing.T) {
	tests := []struct {
		name    string
		profile settings.PerformanceProfile
		hw      settings.HardwareSpecs
		want    int
	}{
		{"balanced is held to its floor on a small machine", settings.ProfileBalanced, machines[2].hw, 16},
		{"balanced follows free memory", settings.ProfileBalanced, settings.HardwareSpecs{LogicalCPUs: 4, TotalRAMMB: 16384, AvailableRAMMB: 8192}, 32},
		{"balanced is held to its ceiling", settings.ProfileBalanced, machines[3].hw, 64},
		{"maximum is held to its floor on a small machine", settings.ProfileMaximum, machines[2].hw, 32},
		{"maximum follows free memory", settings.ProfileMaximum, machines[1].hw, 70},
		{"maximum is held to its ceiling", settings.ProfileMaximum, machines[3].hw, 256},
		{"free memory falls back to half the installed memory", settings.ProfileMaximum, settings.HardwareSpecs{LogicalCPUs: 4, TotalRAMMB: 16384}, 64},
		{"unknown memory falls back to a documented constant", settings.ProfileMaximum, machines[4].hw, 32},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := settings.ResolveCompute(tt.profile, tt.hw, settings.PerformanceCustom{})
			if res.EmbeddingBatchSize != tt.want {
				t.Errorf("batch = %d, want %d", res.EmbeddingBatchSize, tt.want)
			}
		})
	}
}

func TestResolveCompute_TreatsAnUnreportedMachineAsOneCPU(t *testing.T) {
	res := settings.ResolveCompute(settings.ProfileMaximum, settings.HardwareSpecs{}, settings.PerformanceCustom{})

	if res.CoreBudget != 1 || res.OcrThreads != 1 || res.OcrSessions != 1 {
		t.Errorf("resolved %+v, want a budget of one", res)
	}
}

func TestResolveCompute_KeepsBothCustomOcrCountsAsSaved(t *testing.T) {
	custom := settings.PerformanceCustom{EmbeddingBatchSize: 96, OcrThreads: 6, OcrSessions: 3, LlmConcurrency: 5}

	res := settings.ResolveCompute(settings.ProfileCustom, machines[2].hw, custom)

	if res.EmbeddingBatchSize != 96 || res.OcrThreads != 6 || res.OcrSessions != 3 || res.LlmConcurrency != 5 {
		t.Errorf("resolved %+v, want the written values", res)
	}
}

func TestResolveCompute_DerivesTheUnsetCustomOcrCountFromTheBudget(t *testing.T) {
	hw := machines[0].hw

	threadsOnly := settings.ResolveCompute(settings.ProfileCustom, hw, settings.PerformanceCustom{OcrThreads: 2})
	sessionsOnly := settings.ResolveCompute(settings.ProfileCustom, hw, settings.PerformanceCustom{OcrSessions: 2})
	neither := settings.ResolveCompute(settings.ProfileCustom, hw, settings.PerformanceCustom{})

	if threadsOnly.OcrSessions != 8 {
		t.Errorf("sessions = %d, want 8 so that 2 threads fill the budget of 16", threadsOnly.OcrSessions)
	}
	if sessionsOnly.OcrThreads != 8 {
		t.Errorf("threads = %d, want 8 so that 2 sessions fill the budget of 16", sessionsOnly.OcrThreads)
	}
	if neither.OcrThreads*neither.OcrSessions != 16 {
		t.Errorf("threads x sessions = %d, want the budget of 16", neither.OcrThreads*neither.OcrSessions)
	}
}

func TestResolveCompute_ClampsCustomValuesToTheirBounds(t *testing.T) {
	custom := settings.PerformanceCustom{EmbeddingBatchSize: 9999, OcrThreads: 500, OcrSessions: 500, LlmConcurrency: 0}

	res := settings.ResolveCompute(settings.ProfileCustom, machines[0].hw, custom)

	if res.EmbeddingBatchSize != 512 || res.OcrThreads != 64 || res.OcrSessions != 16 || res.LlmConcurrency != 1 {
		t.Errorf("resolved %+v, want batch 512, threads 64, sessions 16, llm 1", res)
	}
}

func TestResolveCompute_FallsBackToTheBalancedPlanForAnUnknownProfile(t *testing.T) {
	res := settings.ResolveCompute("turbo", machines[0].hw, settings.PerformanceCustom{})

	if res.Profile != settings.ProfileBalanced {
		t.Errorf("profile = %q, want balanced", res.Profile)
	}
}

func TestDetectHardware_ReportsConsistentCounts(t *testing.T) {
	hw := settings.DetectHardware()

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

func TestResolveCompute_DerivesTheMemoryLimitFromInstalledMemoryAndProfile(t *testing.T) {
	cases := []struct {
		name    string
		profile settings.PerformanceProfile
		hw      settings.HardwareSpecs
		want    int64
	}{
		{"eco on a Ryzen with 32 GB", settings.ProfileEco, settings.HardwareSpecs{LogicalCPUs: 16, TotalRAMMB: 32768}, 8192},
		{"balanced on a Ryzen with 32 GB", settings.ProfileBalanced, settings.HardwareSpecs{LogicalCPUs: 16, TotalRAMMB: 32768}, 16384},
		{"maximum on a Ryzen with 32 GB", settings.ProfileMaximum, settings.HardwareSpecs{LogicalCPUs: 16, TotalRAMMB: 32768}, 24576},
		{"balanced on an M4 with 24 GB", settings.ProfileBalanced, settings.HardwareSpecs{LogicalCPUs: 10, TotalRAMMB: 24576}, 12288},
		{"maximum on an M4 with 24 GB", settings.ProfileMaximum, settings.HardwareSpecs{LogicalCPUs: 10, TotalRAMMB: 24576}, 18432},
		{"eco on a laptop with 2 GB", settings.ProfileEco, settings.HardwareSpecs{LogicalCPUs: 2, TotalRAMMB: 2048}, 512},
		{"balanced on a laptop with 2 GB", settings.ProfileBalanced, settings.HardwareSpecs{LogicalCPUs: 2, TotalRAMMB: 2048}, 1024},
		{"custom follows balanced", settings.ProfileCustom, settings.HardwareSpecs{LogicalCPUs: 16, TotalRAMMB: 32768}, 16384},
		{"unknown memory sets no limit at eco", settings.ProfileEco, settings.HardwareSpecs{LogicalCPUs: 8}, 0},
		{"unknown memory sets no limit at balanced", settings.ProfileBalanced, settings.HardwareSpecs{LogicalCPUs: 8}, 0},
		{"unknown memory sets no limit at maximum", settings.ProfileMaximum, settings.HardwareSpecs{LogicalCPUs: 8}, 0},
		{"unknown memory sets no limit at custom", settings.ProfileCustom, settings.HardwareSpecs{LogicalCPUs: 8}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := settings.ResolveCompute(c.profile, c.hw, settings.PerformanceCustom{}).MemoryLimitMB
			if got != c.want {
				t.Fatalf("MemoryLimitMB = %d, want %d", got, c.want)
			}
		})
	}
}

func TestResolveCompute_RaisesTheMemoryLimitWithTheProfile(t *testing.T) {
	hw := settings.HardwareSpecs{LogicalCPUs: 8, TotalRAMMB: 16384}
	eco := settings.ResolveCompute(settings.ProfileEco, hw, settings.PerformanceCustom{}).MemoryLimitMB
	balanced := settings.ResolveCompute(settings.ProfileBalanced, hw, settings.PerformanceCustom{}).MemoryLimitMB
	maximum := settings.ResolveCompute(settings.ProfileMaximum, hw, settings.PerformanceCustom{}).MemoryLimitMB
	if !(eco < balanced && balanced < maximum && maximum < hw.TotalRAMMB) {
		t.Fatalf("limits eco=%d balanced=%d maximum=%d are not ordered below %d", eco, balanced, maximum, hw.TotalRAMMB)
	}
}
