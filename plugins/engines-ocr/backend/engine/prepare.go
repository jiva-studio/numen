package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/onnx"
)

// NewLibrary returns the ONNX Runtime library a page is read through, which binds the go-ocr runtime
// when it opens.
func NewLibrary() *onnx.Library {
	return onnx.NewLibrary(bindRuntime)
}

// bindRuntime binds the go-ocr runtime to the library at libPath. paddle.NewEngine binds the
// runtime before it reads the dictionary, so a dictionary that does not exist shows the binding
// succeeded; any other error is a failed binding. The missing dictionary lies in a private
// directory made for the probe, so no other user can plant a file there.
func bindRuntime(libPath string) (err error) {
	probe, err := os.MkdirTemp("", "numen-ocr-bind-probe-")
	if err != nil {
		return fmt.Errorf("create the probe directory: %w", err)
	}
	defer func() { err = errors.Join(err, os.RemoveAll(probe)) }()
	_, bindErr := paddle.NewEngine(paddle.Config{
		OnnxRuntimeLibPath: libPath,
		DetModelPath:       "", RecModelPath: "", DictPath: filepath.Join(probe, "missing-dictionary.txt"),
		UseCuda: false, NumThreads: 0, ThreadCount: 0, DetMaxSideLen: 0, DetOutsideExpandPix: 0,
		RecHeight: 0, RecModelNumClasses: 0, HeatmapThreshold: 0,
	})
	if bindErr == nil || errors.Is(bindErr, fs.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("bind the runtime at %s: %w", libPath, bindErr)
}

// IsPrepared reports whether the library is open.
func IsPrepared(library *onnx.Library) bool { return library.IsAvailable("") }

// Prepare makes the runtime a page is read through.
func Prepare(ctx context.Context, library *onnx.Library, cfg Config) error {
	cfg.ShouldDownload = false
	if _, _, err := library.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir)); err != nil {
		return err
	}
	return nil
}
