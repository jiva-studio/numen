package onnx

import (
	"fmt"
	"runtime"
	"strings"
)

// Provider is an execution provider backend for ONNX Runtime.
type Provider string

const (
	// ProviderAuto selects the best hardware accelerator available on the
	// machine (CoreML on Apple Silicon, CUDA on supported Linux/Windows GPUs,
	// DirectML on Windows DirectX 12 GPUs) and falls back to CPU if unavailable.
	ProviderAuto Provider = "auto"

	// ProviderCPU runs inference on CPU using the configured intra-op threads.
	ProviderCPU Provider = "cpu"

	// ProviderCoreML runs inference on Apple Neural Engine and Apple GPU via CoreML.
	ProviderCoreML Provider = "coreml"

	// ProviderCUDA runs inference on NVIDIA GPUs via CUDA.
	ProviderCUDA Provider = "cuda"

	// ProviderDirectML runs inference on DirectX 12 GPUs on Windows.
	ProviderDirectML Provider = "directml"

	// ProviderROCm runs inference on AMD GPUs on Linux.
	ProviderROCm Provider = "rocm"
)

// ParseProvider converts a string into a valid Provider.
func ParseProvider(name string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "auto":
		return ProviderAuto, nil
	case "cpu":
		return ProviderCPU, nil
	case "coreml", "core-ml":
		return ProviderCoreML, nil
	case "cuda":
		return ProviderCUDA, nil
	case "directml", "direct-ml", "dml":
		return ProviderDirectML, nil
	case "rocm":
		return ProviderROCm, nil
	default:
		return ProviderCPU, fmt.Errorf("unknown execution provider %q, expected one of: auto, cpu, coreml, cuda, directml, rocm", name)
	}
}

// ComputeUnits names the hardware CoreML may place a model on.
type ComputeUnits string

const (
	// ComputeUnitsDefault leaves the choice to the runtime's setup, which is
	// ComputeUnitsAll.
	ComputeUnitsDefault ComputeUnits = ""

	// ComputeUnitsAll places the model on the CPU, the GPU and the Neural Engine.
	ComputeUnitsAll ComputeUnits = "ALL"

	// ComputeUnitsCPUAndNeuralEngine places the model on the CPU and the Neural Engine.
	ComputeUnitsCPUAndNeuralEngine ComputeUnits = "CPUAndNeuralEngine"

	// ComputeUnitsCPUAndGPU places the model on the CPU and the GPU.
	ComputeUnitsCPUAndGPU ComputeUnits = "CPUAndGPU"

	// ComputeUnitsCPUOnly places the model on the CPU.
	ComputeUnitsCPUOnly ComputeUnits = "CPUOnly"
)

// ParseComputeUnits converts a string into valid ComputeUnits. The spelling is
// the runtime's own and the match ignores case.
func ParseComputeUnits(name string) (ComputeUnits, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		// gomutants:disable-next-line RETURN_ZERO reason="the mutant replaces only ComputeUnitsDefault with *new(ComputeUnits), the same empty string; the error value is untouched"
		return ComputeUnitsDefault, nil
	}
	for _, units := range []ComputeUnits{
		ComputeUnitsAll, ComputeUnitsCPUAndNeuralEngine, ComputeUnitsCPUAndGPU, ComputeUnitsCPUOnly,
	} {
		if strings.EqualFold(clean, string(units)) {
			return units, nil
		}
	}
	// gomutants:disable-next-line RETURN_ZERO reason="the mutant replaces only ComputeUnitsDefault with *new(ComputeUnits), the same empty string; the error value is untouched"
	return ComputeUnitsDefault, fmt.Errorf("unknown compute units %q, expected one of: ALL, CPUAndNeuralEngine, CPUAndGPU, CPUOnly", name)
}

// SessionSettings configures how an ONNX session is initialized. Only the
// CoreML provider reads ComputeUnits.
type SessionSettings struct {
	Provider     Provider
	Threads      int
	ComputeUnits ComputeUnits
}

// GetDefaultThreads returns the default intra-op thread count for ONNX sessions.
func GetDefaultThreads() int {
	return min(runtime.GOMAXPROCS(0), 8)
}
