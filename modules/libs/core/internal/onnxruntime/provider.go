package onnxruntime

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	ort "github.com/getcharzp/onnxruntime_purego"
)

func init() {
	Register(func(at string) {
		_ = initGraphOptimization(at)
	})
}

var (
	optLevelMu                        sync.Mutex
	setSessionGraphOptimizationLevel func(options uintptr, level uint32) uintptr
)

type ortApiBase struct {
	GetAPI           uintptr
	GetVersionString uintptr
}

type ortApi struct {
	_                                [23]uintptr
	SetSessionGraphOptimizationLevel uintptr
}

func initGraphOptimization(at string) error {
	optLevelMu.Lock()
	defer optLevelMu.Unlock()
	if setSessionGraphOptimizationLevel != nil {
		return nil
	}
	if at == "" {
		return fmt.Errorf("onnx runtime not loaded")
	}

	handle, err := loadLibrary(at)
	if err != nil {
		return err
	}

	var ortGetApiBase func() *ortApiBase
	purego.RegisterLibFunc(&ortGetApiBase, handle, "OrtGetApiBase")
	if ortGetApiBase == nil {
		return fmt.Errorf("symbol OrtGetApiBase not found")
	}

	apiBase := ortGetApiBase()
	if apiBase == nil {
		return fmt.Errorf("OrtGetApiBase returned nil")
	}

	var getAPI func(uint32) *ortApi
	purego.RegisterFunc(&getAPI, apiBase.GetAPI)
	api := getAPI(23)
	if api == nil || api.SetSessionGraphOptimizationLevel == 0 {
		return fmt.Errorf("failed to get OrtApi or SetSessionGraphOptimizationLevel")
	}

	var setOptLevel func(options uintptr, level uint32) uintptr
	purego.RegisterFunc(&setOptLevel, api.SetSessionGraphOptimizationLevel)
	setSessionGraphOptimizationLevel = setOptLevel
	return nil
}

func setGraphOptimizationLevel(opts *ort.SessionOptions, level uint32) error {
	optLevelMu.Lock()
	fn := setSessionGraphOptimizationLevel
	optLevelMu.Unlock()

	if fn == nil {
		held.mu.Lock()
		at := held.at
		held.mu.Unlock()

		if at == "" {
			return fmt.Errorf("onnx runtime not loaded")
		}
		if err := initGraphOptimization(at); err != nil {
			return err
		}
		optLevelMu.Lock()
		fn = setSessionGraphOptimizationLevel
		optLevelMu.Unlock()
	}

	optHandle := *(*uintptr)(unsafe.Pointer(opts))
	if optHandle == 0 {
		return fmt.Errorf("invalid session options handle")
	}

	status := fn(optHandle, level)
	if status != 0 {
		return fmt.Errorf("OrtSetSessionGraphOptimizationLevel returned status %d", status)
	}
	return nil
}

// Provider is an execution provider backend for ONNX Runtime.
type Provider string

const (
	// ProviderAuto selects the best hardware accelerator available on the
	// machine (CoreML on Apple Silicon, CUDA on supported Linux/Windows GPUs)
	// and falls back to CPU if unavailable.
	ProviderAuto Provider = "auto"

	// ProviderCPU runs inference on CPU using the configured intra-op threads.
	ProviderCPU Provider = "cpu"

	// ProviderCoreML runs inference on Apple Neural Engine and Apple GPU via CoreML.
	ProviderCoreML Provider = "coreml"

	// ProviderCUDA runs inference on NVIDIA GPUs via CUDA.
	ProviderCUDA Provider = "cuda"

	// ProviderDirectML runs inference on DirectX 12 GPUs on Windows.
	ProviderDirectML Provider = "directml"

	// ProviderROCm runs inference on AMD GPUs on Linux.
	ProviderROCm Provider = "rocm"
)

// ParseProvider converts a string into a valid Provider.
func ParseProvider(name string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "auto":
		return ProviderAuto, nil
	case "cpu":
		return ProviderCPU, nil
	case "coreml", "core-ml":
		return ProviderCoreML, nil
	case "cuda":
		return ProviderCUDA, nil
	case "directml", "direct-ml", "dml":
		return ProviderDirectML, nil
	case "rocm":
		return ProviderROCm, nil
	default:
		return ProviderCPU, fmt.Errorf("unknown execution provider %q, expected one of: auto, cpu, coreml, cuda, directml, rocm", name)
	}
}

// SessionSettings configures how an ONNX session is initialized.
type SessionSettings struct {
	Provider Provider
	Threads  int
}

// NewSessionOptions creates and configures session options according to the
// requested provider, applying hardware acceleration with fallback to CPU.
func NewSessionOptions(engine *ort.Engine, s SessionSettings) (*ort.SessionOptions, Provider, error) {
	opts, err := engine.NewSessionOptions()
	if err != nil {
		return nil, "", err
	}

	threads := s.Threads
	if threads <= 0 {
		threads = 4
	}
	if err := opts.SetIntraOpNumThreads(int32(threads)); err != nil {
		opts.Destroy()
		return nil, "", err
	}

	if err := opts.SetCpuMemArena(true); err != nil {
		opts.Destroy()
		return nil, "", err
	}

	if err := setGraphOptimizationLevel(opts, 99); err != nil {
		opts.Destroy()
		return nil, "", err
	}

	used, err := applyProvider(opts, s.Provider)
	if err != nil {
		if s.Provider != ProviderAuto && s.Provider != "" {
			opts.Destroy()
			return nil, "", fmt.Errorf("failed to enable execution provider %s: %w", s.Provider, err)
		}
		used = ProviderCPU
	}

	return opts, used, nil
}

