package onnx

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	ort "github.com/getcharzp/onnxruntime_purego"
)

type graphOptimization struct {
	mu       sync.Mutex
	setLevel func(options uintptr, level uint32) uintptr
}

type ortAPIBase struct {
	GetAPI           uintptr
	GetVersionString uintptr
}

type ortAPI struct {
	_                                [23]uintptr
	SetSessionGraphOptimizationLevel uintptr
}

func (l *Library) bindGraphOptimization(libPath string) error {
	l.graph.mu.Lock()
	defer l.graph.mu.Unlock()
	if l.graph.setLevel != nil {
		return nil
	}
	if libPath == "" {
		return fmt.Errorf("onnx runtime not loaded")
	}

	handle, err := loadLibrary(libPath)
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
	l.graph.setLevel = setOptLevel
	return nil
}

func (l *Library) setGraphOptimizationLevel(opts *ort.SessionOptions, level uint32) error {
	l.graph.mu.Lock()
	fn := l.graph.setLevel
	l.graph.mu.Unlock()

	if fn == nil {
		_, loadedPath := l.runtime.GetLoaded()

		if err := l.bindGraphOptimization(loadedPath); err != nil {
			return err
		}
		l.graph.mu.Lock()
		fn = l.graph.setLevel
		l.graph.mu.Unlock()
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

// NewSessionOptions creates and configures session options according to the
// requested provider, applying hardware acceleration with fallback to CPU.
func (l *Library) NewSessionOptions(engine *ort.Engine, s SessionSettings) (*ort.SessionOptions, Provider, error) {
	opts, err := engine.NewSessionOptions()
	if err != nil {
		return nil, "", err
	}

	threads := s.Threads
	if threads <= 0 {
		threads = GetDefaultThreads()
	}
	if err := opts.SetIntraOpNumThreads(int32(threads)); err != nil {
		opts.Destroy()
		return nil, "", err
	}

	if err := opts.SetCpuMemArena(true); err != nil {
		opts.Destroy()
		return nil, "", err
	}

	if err := l.setGraphOptimizationLevel(opts, 99); err != nil {
		opts.Destroy()
		return nil, "", err
	}

	if err := l.disableSpinning(opts); err != nil {
		opts.Destroy()
		return nil, "", err
	}

	if _, err := ParseComputeUnits(string(s.ComputeUnits)); err != nil {
		opts.Destroy()
		return nil, "", err
	}

	used, err := l.applyProvider(opts, s.Provider, s.ComputeUnits, s.ModelPath)
	if err != nil {
		if s.Provider != ProviderAuto && s.Provider != "" {
			opts.Destroy()
			return nil, "", fmt.Errorf("failed to enable execution provider %s: %w", s.Provider, err)
		}
		used = ProviderCPU
	}

	return opts, used, nil
}
