package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/engine/onnx"
)

// NewLibrary returns the ONNX Runtime library a page is read through, which binds the go-ocr runtime
// when it opens.
func NewLibrary() *onnx.Library {
	return onnx.NewLibrary(bindRuntime)
}

func bindRuntime(libPath string) error {
	probe, err := os.MkdirTemp("", "numen-ocr-bind-probe-")
	if err != nil {
		return fmt.Errorf("create the probe directory: %w", err)
	}
	_, bindErr := paddle.NewEngine(newProbeConfig(libPath, probe))
	return errors.Join(checkBindError(libPath, bindErr), os.RemoveAll(probe))
}

func checkBindError(libPath string, bindErr error) error {
	if bindErr == nil || errors.Is(bindErr, fs.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("bind the runtime at %s: %w", libPath, bindErr)
}

func newProbeConfig(libPath, probeDir string) paddle.Config {
	return paddle.Config{
		OnnxRuntimeLibPath: libPath,
		DetModelPath:       filepath.Join(probeDir, "missing-detection.onnx"),
		RecModelPath:       filepath.Join(probeDir, "missing-recognition.onnx"),
		DictPath:           filepath.Join(probeDir, "missing-dictionary.txt"),
		UseCuda:            false, NumThreads: 0, ThreadCount: 0, DetMaxSideLen: 0, DetOutsideExpandPix: 0,
		RecHeight: 0, RecModelNumClasses: 0, HeatmapThreshold: 0,
	}
}
