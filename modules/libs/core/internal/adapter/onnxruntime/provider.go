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

	// ProviderCoreML runs inference on the Apple Neural Engine and GPU.
	ProviderCoreML Provider = "coreml"
)

// ParseProvider converts a name into a Provider. The empty name is the CPU. The
// cuda provider is named in the error: the published archives are the CPU
// builds, and a GPU build is not provisioned.
func ParseProvider(name string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "cpu":
		return ProviderCPU, nil
	case "coreml":
		return ProviderCoreML, nil
	case "cuda":
		return "", fmt.Errorf("the cuda execution provider needs the GPU build of the runtime, which the host does not provision")
	}
	return "", fmt.Errorf("unknown execution provider %q, expected one of: cpu, coreml", name)
}

// check returns an error when the provider does not run on the platform.
func (p Provider) check(goos, goarch string) error {
	if p == ProviderCoreML && (goos != "darwin" || goarch != "arm64") {
		return fmt.Errorf("the coreml execution provider is only available on darwin/arm64, not %s/%s", goos, goarch)
	}
	return nil
}
