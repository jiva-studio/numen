package onnxruntime

import (
	"slices"
	"testing"
)

func TestComputeUnitsAreParsedBySpelling(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  ComputeUnits
		err   bool
	}{
		{"", ComputeUnitsDefault, false},
		{"  ", ComputeUnitsDefault, false},
		{"ALL", ComputeUnitsAll, false},
		{"all", ComputeUnitsAll, false},
		{"CPUAndNeuralEngine", ComputeUnitsCPUAndNeuralEngine, false},
		{"cpuandneuralengine", ComputeUnitsCPUAndNeuralEngine, false},
		{"CPUAndGPU", ComputeUnitsCPUAndGPU, false},
		{"CPUOnly", ComputeUnitsCPUOnly, false},
		{"NeuralEngine", ComputeUnitsDefault, true},
	} {
		got, err := ParseComputeUnits(tc.input)
		if (err != nil) != tc.err || got != tc.want {
			t.Errorf("ParseComputeUnits(%q) = %q, %v; want %q, error %v", tc.input, got, err, tc.want, tc.err)
		}
	}
}

func TestCoreMLOptionsCarryTheComputeUnitsAskedFor(t *testing.T) {
	for units, want := range map[ComputeUnits]string{
		ComputeUnitsDefault:            "ALL",
		ComputeUnitsAll:                "ALL",
		ComputeUnitsCPUAndNeuralEngine: "CPUAndNeuralEngine",
		ComputeUnitsCPUAndGPU:          "CPUAndGPU",
		ComputeUnitsCPUOnly:            "CPUOnly",
	} {
		got := getCoreMLOptions("/cache", units)
		wantAll := [][2]string{
			{"ModelCacheDirectory", "/cache"},
			{"EnableOnSubgraphs", "1"},
			{"CreateMLProgram", "1"},
			{"MLComputeUnits", want},
		}
		if !slices.Equal(got, wantAll) {
			t.Errorf("getCoreMLOptions(%q) = %v, want %v", units, got, wantAll)
		}
	}
}
