package engine

import (
	"context"
	"sync/atomic"

	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/modules/plugins/ocr/backend/onnxruntime"
)

func init() {
	onnxruntime.Register(func(at string) {
		_, _ = paddle.NewEngine(paddle.Config{OnnxRuntimeLibPath: at})
	})
}

var prepared atomic.Bool

// IsPrepared says whether this process made the runtime.
func IsPrepared() bool { return prepared.Load() }

// Prepare makes the runtime a page is read through.
func Prepare(_ context.Context, cfg Config) error {
	cfg.ShouldDownload = false
	if _, _, err := onnxruntime.Open(cfg.Runtime); err != nil {
		return err
	}
	prepared.Store(true)
	return nil
}
