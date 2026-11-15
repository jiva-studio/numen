package onnx

import (
	"fmt"
	"slices"
	"strings"
)

func checkProvider(libPath string, provider Provider) error {
	name, isGPU := getRuntimeName(provider)
	if !isGPU {
		return nil
	}
	available, err := listProviders(libPath)
	if err != nil {
		return fmt.Errorf("list the providers of the onnx runtime %s: %w", libPath, err)
	}
	return checkProviders(provider, name, libPath, available)
}

func getRuntimeName(provider Provider) (string, bool) {
	parsed, err := ParseProvider(string(provider))
	if err != nil {
		return "", false
	}
	switch parsed {
	case ProviderCUDA:
		return "CUDAExecutionProvider", true
	case ProviderROCm:
		return "ROCMExecutionProvider", true
	case ProviderDirectML:
		return "DmlExecutionProvider", true
	case ProviderAuto, ProviderCPU, ProviderCoreML:
		return "", false
	}
	return "", false
}

func checkProviders(provider Provider, name, libPath string, available []string) error {
	if available == nil || slices.Contains(available, name) {
		return nil
	}
	return fmt.Errorf("the %s provider needs a GPU build of the onnx runtime, and %s is built with %s: install one and name it in %s, or use the cpu provider",
		provider, libPath, strings.Join(available, ", "), EnvLibrary)
}
