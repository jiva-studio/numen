//go:build !darwin && !windows

package onnxruntime

import (
	"fmt"

	ort "github.com/getcharzp/onnxruntime_purego"
)

func applyProvider(opts *ort.SessionOptions, requested Provider) (Provider, error) {
	switch requested {
	case ProviderCPU:
		return ProviderCPU, nil
	case ProviderCUDA:
		if err := opts.EnableCUDA(); err != nil {
			return ProviderCPU, err
		}
		return ProviderCUDA, nil
	case ProviderCoreML:
		return ProviderCPU, fmt.Errorf("coreml execution provider is only available on darwin/arm64")
	case ProviderDirectML:
		return ProviderCPU, fmt.Errorf("directml execution provider is only available on windows")
	case ProviderAuto:
		if err := opts.EnableCUDA(); err == nil {
			return ProviderCUDA, nil
		}
		return ProviderCPU, nil
	default:
		return ProviderCPU, nil
	}
}
