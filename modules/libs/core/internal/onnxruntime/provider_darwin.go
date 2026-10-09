//go:build darwin

package onnxruntime

import (
	"fmt"
	"runtime"
	"syscall"
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
	case ProviderDirectML:
		return ProviderCPU, fmt.Errorf("directml execution provider is only available on windows")
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

	optHandle := *(*uintptr)(unsafe.Pointer(opts))
	if optHandle == 0 {
		return fmt.Errorf("invalid session options handle")
	}

	if appendCoreMLCached(handle, optHandle) {
		return nil
	}

	var appendCoreML func(options uintptr, flags uint32) uintptr
	purego.RegisterLibFunc(&appendCoreML, handle, "OrtSessionOptionsAppendExecutionProvider_CoreML")
	if appendCoreML == nil {
		return fmt.Errorf("symbol OrtSessionOptionsAppendExecutionProvider_CoreML not found")
	}

	status := appendCoreML(optHandle, coreMLFlagEnableOnSubgraph)
	if status != 0 {
		return fmt.Errorf("OrtSessionOptionsAppendExecutionProvider_CoreML returned status %d", status)
	}
	return nil
}

// ortAPIKeyed reaches the runtime's keyed execution-provider entry point.
type ortAPIKeyed struct {
	_                                     [216]uintptr
	SessionOptionsAppendExecutionProvider uintptr
}

// appendCoreMLCached appends CoreML with compiled models kept in the cache
// directory. It reports false when anything it needs is missing, and leaves the
// options untouched.
func appendCoreMLCached(handle, optHandle uintptr) bool {
	dir, err := GetCoreMLCacheDir()
	if err != nil {
		return false
	}

	var getAPIBase func() *ortAPIBase
	purego.RegisterLibFunc(&getAPIBase, handle, "OrtGetApiBase")
	if getAPIBase == nil {
		return false
	}
	base := getAPIBase()
	if base == nil {
		return false
	}
	var getAPI func(uint32) *ortAPIKeyed
	purego.RegisterFunc(&getAPI, base.GetAPI)
	api := getAPI(23)
	if api == nil || api.SessionOptionsAppendExecutionProvider == 0 {
		return false
	}

	var appendProvider func(options uintptr, name *byte, keys, values **byte, count uintptr) uintptr
	purego.RegisterFunc(&appendProvider, api.SessionOptionsAppendExecutionProvider)

	name, err := syscall.BytePtrFromString("CoreML")
	if err != nil {
		return false
	}
	pairs := [][2]string{
		{"ModelCacheDirectory", dir},
		{"EnableOnSubgraphs", "1"},
	}
	keys := make([]*byte, len(pairs))
	values := make([]*byte, len(pairs))
	for i, pair := range pairs {
		if keys[i], err = syscall.BytePtrFromString(pair[0]); err != nil {
			return false
		}
		if values[i], err = syscall.BytePtrFromString(pair[1]); err != nil {
			return false
		}
	}

	status := appendProvider(optHandle, name, &keys[0], &values[0], uintptr(len(pairs)))
	runtime.KeepAlive(keys)
	runtime.KeepAlive(values)
	return status == 0
}
