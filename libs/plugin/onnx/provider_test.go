package onnx

import (
	"strings"
	"testing"
)

func TestAGPUProviderTheLibraryLacksIsRefused(t *testing.T) {
	cpuBuild := []string{"CoreMLExecutionProvider", "CPUExecutionProvider"}
	for _, provider := range []string{"cuda", "ROCm", "directml", "dml"} {
		name, isGPU := getRuntimeName(provider)
		if !isGPU {
			t.Fatalf("%s is not a GPU provider", provider)
		}
		err := checkProviders(provider, name, "/models/libonnxruntime.so", cpuBuild)
		if err == nil || !strings.Contains(err.Error(), "GPU build") || !strings.Contains(err.Error(), EnvLibrary) {
			t.Errorf("%s on a CPU build: error = %v", provider, err)
		}
	}
}

func TestAGPUProviderTheLibraryHoldsPasses(t *testing.T) {
	gpuBuild := []string{"CUDAExecutionProvider", "CPUExecutionProvider"}
	name, _ := getRuntimeName("cuda")
	if err := checkProviders("cuda", name, "/models/libonnxruntime.so", gpuBuild); err != nil {
		t.Errorf("cuda on a CUDA build: %v", err)
	}
}

func TestAProviderOtherThanAGPUOneIsNotChecked(t *testing.T) {
	for _, provider := range []string{"", "auto", "cpu", "coreml"} {
		if err := CheckProvider("/no/such/library", provider); err != nil {
			t.Errorf("%q: %v", provider, err)
		}
	}
}

func TestALibraryThatWasNotOpenedPasses(t *testing.T) {
	name, _ := getRuntimeName("cuda")
	if err := checkProviders("cuda", name, "/models/libonnxruntime.so", nil); err != nil {
		t.Errorf("an unopened library: %v", err)
	}
}
