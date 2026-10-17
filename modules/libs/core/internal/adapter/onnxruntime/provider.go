package onnxruntime

import (
	"fmt"
	"strings"
)

// Provider is the execution provider a plugin opens its sessions on.
type Provider string

const (
	// ProviderCPU runs inference on the CPU.
	ProviderCPU Provider = "cpu"

	// ProviderCUDA runs inference on NVIDIA GPUs.
	ProviderCUDA Provider = "cuda"

	// ProviderCoreML runs inference on the Apple Neural Engine and GPU.
	ProviderCoreML Provider = "coreml"
)

// ParseProvider converts a name into a Provider. The empty name is the CPU.
func ParseProvider(name string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "cpu":
		return ProviderCPU, nil
	case "cuda":
		return ProviderCUDA, nil
	case "coreml":
		return ProviderCoreML, nil
	}
	return "", fmt.Errorf("unknown execution provider %q, expected one of: cpu, cuda, coreml", name)
}

// check returns an error when the provider does not run on the platform.
func (p Provider) check(goos, goarch string) error {
	switch p {
	case ProviderCPU:
	case ProviderCUDA:
		if goos != "linux" && goos != "windows" {
			return fmt.Errorf("the cuda execution provider is not available on %s", goos)
		}
	case ProviderCoreML:
		if goos != "darwin" || goarch != "arm64" {
			return fmt.Errorf("the coreml execution provider is only available on darwin/arm64, not %s/%s", goos, goarch)
		}
	}
	return nil
}
