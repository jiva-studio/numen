package onnx

import "testing"

func TestParseProviderAcceptsEverySpelling(t *testing.T) {
	for name, want := range map[string]Provider{
		"": ProviderAuto, " Auto ": ProviderAuto, "cpu": ProviderCPU,
		"coreml": ProviderCoreML, "core-ml": ProviderCoreML, "CUDA": ProviderCUDA,
		"directml": ProviderDirectML, "direct-ml": ProviderDirectML, "dml": ProviderDirectML,
		"rocm": ProviderROCm,
	} {
		if got, err := ParseProvider(name); err != nil || got != want {
			t.Errorf("ParseProvider(%q) = %q, %v, want %q", name, got, err, want)
		}
	}
}

func TestParseProviderRejectsAnUnknownNameAndFallsBackToTheCPU(t *testing.T) {
	got, err := ParseProvider("tpu")
	if err == nil || got != ProviderCPU {
		t.Errorf("ParseProvider(tpu) = %q, %v, want cpu and an error", got, err)
	}
}

func TestParseComputeUnitsIgnoresCase(t *testing.T) {
	for name, want := range map[string]ComputeUnits{
		"": ComputeUnitsDefault, " all ": ComputeUnitsAll, "cpuandneuralengine": ComputeUnitsCPUAndNeuralEngine,
		"CPUAndGPU": ComputeUnitsCPUAndGPU, "cpuonly": ComputeUnitsCPUOnly,
	} {
		if got, err := ParseComputeUnits(name); err != nil || got != want {
			t.Errorf("ParseComputeUnits(%q) = %q, %v, want %q", name, got, err, want)
		}
	}
	if got, err := ParseComputeUnits("GPUOnly"); err == nil || got != ComputeUnitsDefault {
		t.Errorf("ParseComputeUnits(GPUOnly) = %q, %v, want the default and an error", got, err)
	}
}

func TestGetDefaultThreadsIsAtMostEight(t *testing.T) {
	if got := GetDefaultThreads(); got < 1 || got > 8 {
		t.Errorf("GetDefaultThreads() = %d, want 1 to 8", got)
	}
}
