package onnxruntime

import (
	"fmt"
	"strings"
)

// Provider is the execution provider a plugin opens its sessions on. The plugin
// applies it; the host passes the name on and refuses a GPU provider for the
// pinned CPU archive.
type Provider string

const (
	// ProviderAuto lets the plugin pick the best accelerator it can enable and fall back to the CPU.
	ProviderAuto Provider = "auto"

	// ProviderCPU runs inference on the CPU.
	ProviderCPU Provider = "cpu"

	// ProviderCoreML runs inference on the Apple Neural Engine and GPU.
	ProviderCoreML Provider = "coreml"

	// ProviderCUDA runs inference on NVIDIA GPUs.
	ProviderCUDA Provider = "cuda"

	// ProviderDirectML runs inference on DirectX 12 GPUs on Windows.
	ProviderDirectML Provider = "directml"

	// ProviderROCm runs inference on AMD GPUs on Linux.
	ProviderROCm Provider = "rocm"
)

// ParseProvider converts a name into a Provider. The empty name is auto.
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
	}
	return "", fmt.Errorf("unknown execution provider %q, expected one of: auto, cpu, coreml, cuda, directml, rocm", name)
}

// isGPU reports whether the provider needs a GPU build of the runtime.
func (p Provider) isGPU() bool {
	return p == ProviderCUDA || p == ProviderDirectML || p == ProviderROCm
}
