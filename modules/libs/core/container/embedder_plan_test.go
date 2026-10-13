package container

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed/openai"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/onnxruntime"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func TestTheEmbedderIsGivenTheThreadBudgetAndWorkersOfThePlan(t *testing.T) {
	machines := map[string]settings.HardwareSpecs{
		"sixteen threads on eight cores": {LogicalCPUs: 16, PhysicalCPUs: 8, PerformanceCPUs: 8, TotalRAMMB: 32768, AvailableRAMMB: 16384},
		"ten cores, four of them slow":   {LogicalCPUs: 10, PhysicalCPUs: 10, PerformanceCPUs: 4, TotalRAMMB: 16384, AvailableRAMMB: 8192},
		"two cores":                      {LogicalCPUs: 2, PhysicalCPUs: 2, PerformanceCPUs: 2, TotalRAMMB: 4096, AvailableRAMMB: 2048},
		"sixty-four cores":               {LogicalCPUs: 64, PhysicalCPUs: 64, PerformanceCPUs: 64, TotalRAMMB: 262144, AvailableRAMMB: 131072},
	}
	profiles := []settings.PerformanceProfile{
		settings.ProfileEco, settings.ProfileBalanced, settings.ProfileMaximum, settings.ProfileCustom,
	}
	for name, hw := range machines {
		for _, profile := range profiles {
			plan := settings.ResolveCompute(profile, hw, settings.PerformanceCustom{})
			local := Config{ResolvedCompute: plan}.planLocalModel(embed.LocalModel{})
			if got := local.GetThreads(); got != plan.CoreBudget {
				t.Errorf("%s, %s: %d threads, want the budget %d", name, profile, got, plan.CoreBudget)
			}
			if got := local.GetPasses(onnxruntime.ProviderCPU); got != max(plan.Workers, 1) {
				t.Errorf("%s, %s: %d passes, want the workers %d", name, profile, got, plan.Workers)
			}
			if got := local.GetPasses(onnxruntime.ProviderCoreML); got != 1 {
				t.Errorf("%s, %s: %d passes on CoreML, want 1", name, profile, got)
			}
			if local.BatchTexts != plan.EmbeddingBatchSize {
				t.Errorf("%s, %s: batch %d, want %d", name, profile, local.BatchTexts, plan.EmbeddingBatchSize)
			}
		}
	}
}

func TestTheSettingsOfTheModelWinOverThePlan(t *testing.T) {
	plan := settings.ResolvedCompute{CoreBudget: 8, Workers: 4, EmbeddingBatchSize: 16}
	local := Config{ResolvedCompute: plan}.planLocalModel(embed.LocalModel{Threads: 2, Sessions: 1, BatchTexts: 7})
	if local.GetThreads() != 2 || local.GetPasses(onnxruntime.ProviderCPU) != 1 || local.BatchTexts != 7 {
		t.Errorf("got %d threads, %d passes, batch %d", local.GetThreads(), local.GetPasses(onnxruntime.ProviderCPU), local.BatchTexts)
	}
}

func TestAContainerWithoutAPlanLeavesTheModelItsFallbacks(t *testing.T) {
	local := Config{}.planLocalModel(embed.LocalModel{})
	if local.ThreadBudget != 0 || local.PassBudget != 0 || local.BatchTexts != 0 {
		t.Errorf("got %+v", local)
	}
	if local.GetThreads() != onnxruntime.DefaultThreads() {
		t.Errorf("threads %d, want the runtime default", local.GetThreads())
	}
}

func TestTheEmbedderMayHoldAShareOfTheAvailableMemory(t *testing.T) {
	for name, hw := range map[string]settings.HardwareSpecs{
		"two gigabytes":              {LogicalCPUs: 4, PhysicalCPUs: 2, PerformanceCPUs: 2, TotalRAMMB: 2048, AvailableRAMMB: 1024},
		"sixteen gigabytes":          {LogicalCPUs: 16, PhysicalCPUs: 8, PerformanceCPUs: 8, TotalRAMMB: 16384, AvailableRAMMB: 12288},
		"server of 128 logical CPUs": {LogicalCPUs: 128, PhysicalCPUs: 64, PerformanceCPUs: 64, TotalRAMMB: 65536, AvailableRAMMB: 60000},
	} {
		plan := settings.ResolveCompute(settings.ProfileMaximum, hw, settings.PerformanceCustom{})
		local := Config{ResolvedCompute: plan, Hardware: hw}.planLocalModel(embed.LocalModel{})
		want := hw.AvailableRAMMB * bytesPerMB / 100 * 25
		if local.PassMemory != want {
			t.Errorf("%s: the passes may hold %d bytes, want %d", name, local.PassMemory, want)
		}
		if local.PassMemory >= hw.AvailableRAMMB*bytesPerMB {
			t.Errorf("%s: the passes may hold all the available memory", name)
		}
	}
}

func TestAContainerThatKnowsNoMemoryBoundsNothing(t *testing.T) {
	local := Config{ResolvedCompute: settings.ResolvedCompute{Workers: 8}}.planLocalModel(embed.LocalModel{})
	if local.PassMemory != 0 {
		t.Errorf("the passes may hold %d bytes", local.PassMemory)
	}
}

func TestTheBatchOfThePlanReachesAModelThatNamedNone(t *testing.T) {
	for _, batch := range []int{1, 2, 64} {
		local := Config{ResolvedCompute: settings.ResolvedCompute{EmbeddingBatchSize: batch}}.planLocalModel(embed.LocalModel{})
		if local.BatchTexts != batch {
			t.Errorf("a plan of %d texts a pass gave the model %d", batch, local.BatchTexts)
		}
	}
}

func TestABatchThePlanCannotMeanLeavesTheModelItsOwn(t *testing.T) {
	local := Config{ResolvedCompute: settings.ResolvedCompute{EmbeddingBatchSize: -1}}.planLocalModel(embed.LocalModel{})
	if local.BatchTexts != 0 {
		t.Errorf("the model was given %d texts a pass", local.BatchTexts)
	}
}

func TestAServiceIsAskedAsManyAtOnceAsThePlanAllows(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")
	is := port.EmbeddingModel{Name: "m", Dimensions: 8}
	service := embed.ServiceModel{Name: "m", BaseURL: "http://127.0.0.1:1/v1"}
	own, err := openai.New(is, service)
	if err != nil {
		t.Fatal(err)
	}

	for _, allowed := range []int{1, 2, 9} {
		client, err := Config{ResolvedCompute: settings.ResolvedCompute{LlmConcurrency: allowed}}.openServiceClient(is, service)
		if err != nil {
			t.Fatal(err)
		}
		if client.InFlight != allowed {
			t.Errorf("a plan of %d at once gave the client %d", allowed, client.InFlight)
		}
	}
	client, err := Config{}.openServiceClient(is, service)
	if err != nil {
		t.Fatal(err)
	}
	if client.InFlight != own.InFlight {
		t.Errorf("a container without a plan gave the client %d, and it asks %d at once of its own", client.InFlight, own.InFlight)
	}
}

func TestAServiceThatCannotBeBuiltIsNotAskedAnything(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")
	client, err := Config{}.openServiceClient(port.EmbeddingModel{Name: "m", Dimensions: 8}, embed.ServiceModel{Name: "m"})
	if err == nil || client != nil {
		t.Errorf("got %v: %v", client, err)
	}
}
