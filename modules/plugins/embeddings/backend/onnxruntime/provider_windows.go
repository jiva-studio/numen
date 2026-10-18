//go:build windows

package onnxruntime

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
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
	case onnx.ProviderDirectML:
		if err := enableDirectML(opts); err != nil {
			return onnx.ProviderCPU, err
		}
		return onnx.ProviderDirectML, nil
	case onnx.ProviderCoreML:
		return onnx.ProviderCPU, fmt.Errorf("coreml execution provider is only available on darwin/arm64")
	case onnx.ProviderAuto:
		if err := opts.EnableCUDA(); err == nil {
			return onnx.ProviderCUDA, nil
		}
		if err := enableDirectML(opts); err == nil {
			return onnx.ProviderDirectML, nil
		}
		return onnx.ProviderCPU, nil
	default:
		return onnx.ProviderCPU, nil
	}
}

func enableDirectML(opts *ort.SessionOptions) error {
	engine, at := runtimeState.GetLoaded()

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
