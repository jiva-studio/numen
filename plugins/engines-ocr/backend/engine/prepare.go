package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

func init() {
	onnx.Register(bindRuntime)
}

// bindRuntime binds the go-ocr runtime to the library at libPath. paddle.NewEngine binds the
// runtime before it reads the dictionary, so a dictionary that does not exist shows the binding
// succeeded; any other error is a failed binding.
func bindRuntime(libPath string) error {
	_, err := paddle.NewEngine(paddle.Config{
		OnnxRuntimeLibPath: libPath,
		DetModelPath:       "", RecModelPath: "", DictPath: missingDictionaryPath(),
		UseCuda: false, NumThreads: 0, ThreadCount: 0, DetMaxSideLen: 0, DetOutsideExpandPix: 0,
		RecHeight: 0, RecModelNumClasses: 0, HeatmapThreshold: 0,
	})
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("bind the runtime at %s: %w", libPath, err)
}

// missingDictionaryPath is a path that names no file.
func missingDictionaryPath() string {
	return filepath.Join(os.TempDir(), "numen-ocr-bind-probe", "missing-dictionary.txt")
}

// IsPrepared reports whether this process created the runtime.
func IsPrepared() bool { return onnx.IsAvailable("") }

// Prepare makes the runtime a page is read through.
func Prepare(ctx context.Context, cfg Config) error {
	cfg.ShouldDownload = false
	if _, _, err := onnx.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir)); err != nil {
		return err
	}
	return nil
}
