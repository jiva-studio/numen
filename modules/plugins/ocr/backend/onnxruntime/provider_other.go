//go:build !darwin && !windows

package onnxruntime

import (
	"fmt"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

func applyProvider(opts *ort.SessionOptions, requested onnx.Provider, _ onnx.ComputeUnits) (onnx.Provider, error) {
	switch requested {
	case onnx.ProviderCPU:
		return onnx.ProviderCPU, nil
	case onnx.ProviderCUDA:
		if err := opts.EnableCUDA(); err != nil {
			return onnx.ProviderCPU, err
		}
		return onnx.ProviderCUDA, nil
	case onnx.ProviderCoreML:
		return onnx.ProviderCPU, fmt.Errorf("coreml execution provider is only available on darwin/arm64")
	case onnx.ProviderDirectML:
		return onnx.ProviderCPU, fmt.Errorf("directml execution provider is only available on windows")
	case onnx.ProviderAuto:
		if err := opts.EnableCUDA(); err == nil {
			return onnx.ProviderCUDA, nil
		}
		return onnx.ProviderCPU, nil
	default:
		return onnx.ProviderCPU, nil
	}
}
