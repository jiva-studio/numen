package recognition

import (
	"context"
	"sync/atomic"

	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// The reading library makes an engine of its own and takes none. It is made
// with the process's runtime, so which of a reading and a transcription opens
// first settles nothing.
func init() {
	onnxruntime.Register(func(at string) {
		// A reader asked for with no models is refused, and the engine it was
		// to read through is made first.
		_, _ = paddle.NewEngine(paddle.Config{OnnxRuntimeLibPath: at})
	})
}

// prepared says whether the runtime a page is read through was made.
var prepared atomic.Bool

// IsPrepared says whether this process made the runtime before it made anything
// else.
func IsPrepared() bool { return prepared.Load() }

// Prepare makes the runtime a page is read through, and is called before a
// window is. Every page this process reads is read through it, and one made
// after a window says nothing about every page it is given.
//
// Nothing is fetched: a machine that holds no runtime says so, and reading is
// what fetches one.
func Prepare(ctx context.Context, cfg Config) error {
	cfg.Download = false
	if _, _, err := onnxruntime.Open(ctx, cfg.settings()); err != nil {
		return err
	}
	prepared.Store(true)
	return nil
}
