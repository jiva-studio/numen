package onnx

import (
	"runtime"
	"testing"
)

func TestParseProviderAcceptsEverySpelling(t *testing.T) {
	for _, tc := range []struct {
		name string
		want Provider
	}{
		{"", ProviderAuto}, {" Auto ", ProviderAuto}, {"cpu", ProviderCPU},
		{"coreml", ProviderCoreML}, {"core-ml", ProviderCoreML}, {"CUDA", ProviderCUDA},
		{"directml", ProviderDirectML}, {"direct-ml", ProviderDirectML}, {"dml", ProviderDirectML},
		{"rocm", ProviderROCm},
	} {
		if got, err := ParseProvider(tc.name); err != nil || got != tc.want {
			t.Errorf("ParseProvider(%q) = %q, %v, want %q", tc.name, got, err, tc.want)
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
	for _, tc := range []struct {
		name string
		want ComputeUnits
	}{
		{"", ComputeUnitsDefault}, {" all ", ComputeUnitsAll}, {"cpuandneuralengine", ComputeUnitsCPUAndNeuralEngine},
		{"CPUAndGPU", ComputeUnitsCPUAndGPU}, {"cpuonly", ComputeUnitsCPUOnly},
	} {
		if got, err := ParseComputeUnits(tc.name); err != nil || got != tc.want {
			t.Errorf("ParseComputeUnits(%q) = %q, %v, want %q", tc.name, got, err, tc.want)
		}
	}
	if got, err := ParseComputeUnits("GPUOnly"); err == nil || got != ComputeUnitsDefault {
		t.Errorf("ParseComputeUnits(GPUOnly) = %q, %v, want the default and an error", got, err)
	}
}

func TestGetDefaultThreadsIsAtMostEight(t *testing.T) {
	prev := runtime.GOMAXPROCS(4)
	defer runtime.GOMAXPROCS(prev)
	if got := GetDefaultThreads(); got != 4 {
		t.Errorf("GetDefaultThreads() with GOMAXPROCS=4 = %d, want 4", got)
	}
	if got := GetDefaultThreads(); got != 4 {
		t.Errorf("second GetDefaultThreads() with GOMAXPROCS=4 = %d, want 4", got)
	}

	runtime.GOMAXPROCS(8)
	if got := GetDefaultThreads(); got != 8 {
		t.Errorf("GetDefaultThreads() with GOMAXPROCS=8 = %d, want 8", got)
	}

	runtime.GOMAXPROCS(16)
	if got := GetDefaultThreads(); got != 8 {
		t.Errorf("GetDefaultThreads() with GOMAXPROCS=16 = %d, want 8", got)
	}
}
