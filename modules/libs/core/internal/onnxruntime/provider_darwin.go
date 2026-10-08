//go:build darwin

package onnxruntime

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
	ort "github.com/getcharzp/onnxruntime_purego"
)

const coreMLFlagEnableOnSubgraph uint32 = 0x002

func applyProvider(opts *ort.SessionOptions, requested Provider) (Provider, error) {
	switch requested {
	case ProviderCPU:
		return ProviderCPU, nil
	case ProviderCoreML, ProviderAuto:
		if runtime.GOARCH == "arm64" {
			if err := enableCoreML(opts); err == nil {
				return ProviderCoreML, nil
			} else if requested == ProviderCoreML {
				return ProviderCPU, err
			}
		}
		return ProviderCPU, nil
	case ProviderCUDA:
		if err := opts.EnableCUDA(); err != nil {
			return ProviderCPU, err
		}
		return ProviderCUDA, nil
	default:
		return ProviderCPU, nil
	}
}

func enableCoreML(opts *ort.SessionOptions) error {
	held.mu.Lock()
	engine := held.engine
	at := held.at
	held.mu.Unlock()

	if engine == nil || at == "" {
		return fmt.Errorf("onnx runtime not loaded")
	}

	handle, err := purego.Dlopen(at, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}

	var appendCoreML func(options uintptr, flags uint32) uintptr
	purego.RegisterLibFunc(&appendCoreML, handle, "OrtSessionOptionsAppendExecutionProvider_CoreML")
	if appendCoreML == nil {
		return fmt.Errorf("symbol OrtSessionOptionsAppendExecutionProvider_CoreML not found")
	}

	optHandle := *(*uintptr)(unsafe.Pointer(opts))
	if optHandle == 0 {
		return fmt.Errorf("invalid session options handle")
	}

	status := appendCoreML(optHandle, coreMLFlagEnableOnSubgraph)
	if status != 0 {
		return fmt.Errorf("OrtSessionOptionsAppendExecutionProvider_CoreML returned status %d", status)
	}
	return nil
}
