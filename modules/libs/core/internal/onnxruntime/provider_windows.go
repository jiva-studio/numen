//go:build windows

package onnxruntime

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
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
	case ProviderDirectML:
		if err := enableDirectML(opts); err != nil {
			return ProviderCPU, err
		}
		return ProviderDirectML, nil
	case ProviderCoreML:
		return ProviderCPU, fmt.Errorf("coreml execution provider is only available on darwin/arm64")
	case ProviderAuto:
		if err := opts.EnableCUDA(); err == nil {
			return ProviderCUDA, nil
		}
		if err := enableDirectML(opts); err == nil {
			return ProviderDirectML, nil
		}
		return ProviderCPU, nil
	default:
		return ProviderCPU, nil
	}
}

func enableDirectML(opts *ort.SessionOptions) error {
	runtimeState.mu.Lock()
	engine := runtimeState.engine
	at := runtimeState.at
	runtimeState.mu.Unlock()

	if engine == nil || at == "" {
		return fmt.Errorf("onnx runtime not loaded")
	}

	handle, err := loadLibrary(at)
	if err != nil {
		return err
	}

	optHandle := *(*uintptr)(unsafe.Pointer(opts))
	if optHandle == 0 {
		return fmt.Errorf("invalid session options handle")
	}

	proc, err := syscall.GetProcAddress(syscall.Handle(handle), "OrtSessionOptionsAppendExecutionProvider_DML")
	if err != nil {
		return fmt.Errorf("symbol OrtSessionOptionsAppendExecutionProvider_DML not found: %w", err)
	}

	var appendDML func(options uintptr, deviceID int32) uintptr
	purego.RegisterFunc(&appendDML, proc)
	if appendDML == nil {
		return fmt.Errorf("failed to register OrtSessionOptionsAppendExecutionProvider_DML")
	}

	status := appendDML(optHandle, 0)
	if status != 0 {
		return fmt.Errorf("OrtSessionOptionsAppendExecutionProvider_DML returned status %d", status)
	}
	return nil
}
