package onnxruntime

import "fmt"

// Provider is the execution provider that a plugin opens its sessions on. The
// plugin enables it; the host passes the name through and rejects a GPU
// provider for the pinned CPU archive.
type Provider string

// The providers, and the spellings that name them. Auto lets the plugin pick the
// best accelerator it can enable and fall back to the CPU.
const (
	ProviderAuto     Provider = "auto"
	ProviderCPU      Provider = "cpu"
	ProviderCoreML   Provider = "coreml"
	ProviderCUDA     Provider = "cuda"
	ProviderDirectML Provider = "directml"
	ProviderROCm     Provider = "rocm"
)

func getProviderNames() map[string]Provider {
	return map[string]Provider{
		"": ProviderAuto, "auto": ProviderAuto, "cpu": ProviderCPU,
		"coreml": ProviderCoreML, "core-ml": ProviderCoreML, "cuda": ProviderCUDA,
		"directml": ProviderDirectML, "direct-ml": ProviderDirectML, "dml": ProviderDirectML,
		"rocm": ProviderROCm,
	}
}

// ParseProvider converts a name into a Provider, ignoring case and surrounding space.
func ParseProvider(name string) (Provider, error) {
	if provider, ok := getProviderNames()[normalize(name)]; ok {
		return provider, nil
	}
	return "", fmt.Errorf("unknown execution provider %q, expected one of: auto, cpu, coreml, cuda, directml, rocm", name)
}

func (p Provider) isGPU() bool {
	return p == ProviderCUDA || p == ProviderDirectML || p == ProviderROCm
}
