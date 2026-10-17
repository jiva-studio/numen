package onnxruntime

import (
	"fmt"
	"os"
	"runtime"
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
	optLevelMu                       sync.Mutex
	setSessionGraphOptimizationLevel func(options uintptr, level uint32) uintptr
)

type ortAPIBase struct {
	GetAPI           uintptr
	GetVersionString uintptr
}

type ortAPI struct {
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

	var ortGetAPIBase func() *ortAPIBase
	purego.RegisterLibFunc(&ortGetAPIBase, handle, "OrtGetApiBase")
	if ortGetAPIBase == nil {
		return fmt.Errorf("symbol OrtGetApiBase not found")
	}

	apiBase := ortGetAPIBase()
	if apiBase == nil {
		return fmt.Errorf("OrtGetApiBase returned nil")
	}

	var getAPI func(uint32) *ortAPI
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
		runtimeState.mu.Lock()
		at := runtimeState.at
		runtimeState.mu.Unlock()

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
	default:
		return ProviderCPU, fmt.Errorf("unknown execution provider %q, expected one of: auto, cpu, coreml, cuda", name)
	}
}

// ComputeUnits names the hardware CoreML may place a model on.
type ComputeUnits string

const (
	// ComputeUnitsDefault leaves the choice to the runtime's setup, which is
	// ComputeUnitsAll.
	ComputeUnitsDefault ComputeUnits = ""

	// ComputeUnitsAll places the model on the CPU, the GPU and the Neural Engine.
	ComputeUnitsAll ComputeUnits = "ALL"

	// ComputeUnitsCPUAndNeuralEngine places the model on the CPU and the Neural Engine.
	ComputeUnitsCPUAndNeuralEngine ComputeUnits = "CPUAndNeuralEngine"

	// ComputeUnitsCPUAndGPU places the model on the CPU and the GPU.
	ComputeUnitsCPUAndGPU ComputeUnits = "CPUAndGPU"

	// ComputeUnitsCPUOnly places the model on the CPU.
	ComputeUnitsCPUOnly ComputeUnits = "CPUOnly"
)

// ParseComputeUnits converts a string into valid ComputeUnits. The spelling is
// the runtime's own and the match ignores case.
func ParseComputeUnits(name string) (ComputeUnits, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		// gomutants:disable-next-line RETURN_ZERO reason="the mutant replaces only ComputeUnitsDefault with *new(ComputeUnits), the same empty string; the error value is untouched"
		return ComputeUnitsDefault, nil
	}
	for _, units := range []ComputeUnits{
		ComputeUnitsAll, ComputeUnitsCPUAndNeuralEngine, ComputeUnitsCPUAndGPU, ComputeUnitsCPUOnly,
	} {
		if strings.EqualFold(trimmed, string(units)) {
			return units, nil
		}
	}
	// gomutants:disable-next-line RETURN_ZERO reason="the mutant replaces only ComputeUnitsDefault with *new(ComputeUnits), the same empty string; the error value is untouched"
	return ComputeUnitsDefault, fmt.Errorf("unknown compute units %q, expected one of: ALL, CPUAndNeuralEngine, CPUAndGPU, CPUOnly", name)
}

// SessionSettings configures how an ONNX session is initialized. Only the
// CoreML provider reads ComputeUnits.
type SessionSettings struct {
	Provider     Provider
	Threads      int
	ComputeUnits ComputeUnits
}

// DefaultThreads is the default intra-op thread count for ONNX sessions.
func DefaultThreads() int {
	return min(runtime.GOMAXPROCS(0), 8)
}

// NewSessionOptions creates and configures session options according to the
// requested provider, applying hardware acceleration with fallback to CPU.
func NewSessionOptions(engine *ort.Engine, s SessionSettings) (*ort.SessionOptions, Provider, error) {
	provider, err := resolveProvider(s.Provider)
	if err != nil {
		return nil, "", err
	}
	opts, err := engine.NewSessionOptions()
	if err != nil {
		return nil, "", err
	}

	threads := s.Threads
	if threads <= 0 {
		threads = DefaultThreads()
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

	if err := disableSpinning(opts); err != nil {
		opts.Destroy()
		return nil, "", err
	}

	if _, err := ParseComputeUnits(string(s.ComputeUnits)); err != nil {
		opts.Destroy()
		return nil, "", err
	}

	used, err := applyProvider(opts, provider, s.ComputeUnits)
	if err != nil {
		if provider != ProviderAuto {
			opts.Destroy()
			return nil, "", fmt.Errorf("failed to enable execution provider %s: %w", provider, err)
		}
		used = ProviderCPU
	}

	return opts, used, nil
}

// resolveProvider returns the requested provider, or the one the host chose
// when none is requested.
func resolveProvider(requested Provider) (Provider, error) {
	if requested != "" {
		return requested, nil
	}
	return ParseProvider(os.Getenv(EnvProvider))
}
