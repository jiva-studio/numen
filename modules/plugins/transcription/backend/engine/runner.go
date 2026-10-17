package engine

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/jiva-studio/numen/modules/plugins/transcription/backend/onnxruntime"
)

var (
	runnerMu sync.Mutex
	ortRun   func(
		session, runOptions, inputNames, inputValues, inputCount, outputNames, outputCount, outputValues uintptr,
	) uintptr
	ortGetTensorMutableData func(value, out uintptr) uintptr
	ortReleaseValue         func(value uintptr)
	ortReleaseStatus        func(status uintptr)
	ortGetErrorCode         func(status uintptr) int32
	ortGetErrorMessage      func(status uintptr) unsafe.Pointer
)

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

func init() {
	onnxruntime.Register(func(at string) {
		_ = initRunner(at)
	})
}

func initRunner(at string) error {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	if ortRun != nil {
		return nil
	}
	if at == "" {
		return fmt.Errorf("onnx runtime not loaded")
	}

	handle, err := onnxruntime.OpenLibrary(at)
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

	purego.RegisterFunc(&ortRun, api.Run)
	purego.RegisterFunc(&ortGetTensorMutableData, api.GetTensorMutableData)
	purego.RegisterFunc(&ortReleaseValue, api.ReleaseValue)
	purego.RegisterFunc(&ortReleaseStatus, api.ReleaseStatus)
	purego.RegisterFunc(&ortGetErrorCode, api.GetErrorCode)
	purego.RegisterFunc(&ortGetErrorMessage, api.GetErrorMessage)
	return nil
}

func checkStatus(status uintptr) error {
	if status == 0 {
		return nil
	}
	defer ortReleaseStatus(status)
	code := ortGetErrorCode(status)
	msgPtr := ortGetErrorMessage(status)
	var msg string
	if msgPtr != nil {
		var b []byte
		for p := (*byte)(msgPtr); *p != 0; p = (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + 1)) {
			b = append(b, *p)
		}
		msg = string(b)
	}
	return fmt.Errorf("onnxruntime error [code %d]: %s", code, msg)
}
