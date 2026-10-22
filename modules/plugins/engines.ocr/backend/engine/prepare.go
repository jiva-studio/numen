package engine

import (
	"context"
	"sync/atomic"

	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

func init() {
	onnx.Register(func(pos string) {
		_, _ = paddle.NewEngine(paddle.Config{OnnxRuntimeLibPath: pos})
	})
}

var prepared atomic.Bool

// IsPrepared says whether this process made the runtime.
func IsPrepared() bool { return prepared.Load() }

// Prepare makes the runtime a page is read through.
func Prepare(ctx context.Context, cfg Config) error {
	cfg.ShouldDownload = false
	if _, _, err := onnx.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir)); err != nil {
		return err
	}
	prepared.Store(true)
	return nil
}
