package ortraw

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/jiva-studio/numen/libs/shared/onnx"
)

// Runner holds the entry points of the runtime a Graph is run through, resolved by Bind from the
// library the runtime opens.
type Runner struct {
	mu     sync.Mutex
	run func(
		session, runOptions, inputNames, inputValues, inputCount, outputNames, outputCount, outputValues uintptr,
	) uintptr
	getTensorMutableData func(value, out uintptr) uintptr
	releaseValue         func(value uintptr)
	releaseStatus        func(status uintptr)
	getErrorCode         func(status uintptr) int32
	getErrorMessage      func(status uintptr) unsafe.Pointer
}

type ortAPIBase struct {
	GetAPI           uintptr
	GetVersionString uintptr
}

type ortAPITable struct {
	_                    uintptr
	GetErrorCode         uintptr
	GetErrorMessage      uintptr
	_                    [6]uintptr
	Run                  uintptr
	_                    [41]uintptr
	GetTensorMutableData uintptr
	_                    [41]uintptr
	ReleaseStatus        uintptr
	_                    [2]uintptr
	ReleaseValue         uintptr
}

// NewRunner returns a runner that has resolved no entry point. It is given to onnx.NewLibrary as
// the hook Bind.
func NewRunner() *Runner {
	return new(Runner)
}

// Bind resolves the entry points of the library at pos.
func (runner *Runner) Bind(pos string) error {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if runner.run != nil {
		return nil
	}
	if pos == "" {
		return fmt.Errorf("onnx runtime not loaded")
	}

	handle, err := onnx.OpenLibrary(pos)
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

	var getAPI func(uint32) *ortAPITable
	purego.RegisterFunc(&getAPI, apiBase.GetAPI)
	api := getAPI(23)
	if api == nil || api.Run == 0 {
		return fmt.Errorf("failed to get OrtApi")
	}

	purego.RegisterFunc(&runner.run, api.Run)
	purego.RegisterFunc(&runner.getTensorMutableData, api.GetTensorMutableData)
	purego.RegisterFunc(&runner.releaseValue, api.ReleaseValue)
	purego.RegisterFunc(&runner.releaseStatus, api.ReleaseStatus)
	purego.RegisterFunc(&runner.getErrorCode, api.GetErrorCode)
	purego.RegisterFunc(&runner.getErrorMessage, api.GetErrorMessage)
	return nil
}

func (runner *Runner) checkStatus(status uintptr) error {
	if status == 0 {
		return nil
	}
	defer runner.releaseStatus(status)
	code := runner.getErrorCode(status)
	msgPtr := runner.getErrorMessage(status)
	var msg string
	if msgPtr != nil {
		var b []byte
		for p := (*byte)(msgPtr); *p != 0; p = (*byte)(unsafe.Add(unsafe.Pointer(p), 1)) {
			b = append(b, *p)
		}
		msg = string(b)
	}
	return fmt.Errorf("onnxruntime error [code %d]: %s", code, msg)
}

// ObserveRelease calls observe with every value the runtime releases, and
// returns the function that stops it.
func (runner *Runner) ObserveRelease(observe func(value uintptr)) (restore func()) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	original := runner.releaseValue
	if original == nil {
		return func() {}
	}
	runner.releaseValue = func(value uintptr) {
		observe(value)
		original(value)
	}
	return func() {
		runner.mu.Lock()
		defer runner.mu.Unlock()
		runner.releaseValue = original
	}
}
