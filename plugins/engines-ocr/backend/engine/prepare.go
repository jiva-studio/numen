package engine

import (
	"context"

	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

func init() {
	onnx.Register(func(pos string) {
		_, _ = paddle.NewEngine(paddle.Config{OnnxRuntimeLibPath: pos})
	})
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
