package recognition

import (
	"context"
	"sync/atomic"

	"github.com/getcharzp/go-ocr/paddle"
)

// standing says whether the runtime a page is read through was made.
var standing atomic.Bool

// Prepared says whether this process made the runtime before it made anything
// else.
func Prepared() bool { return standing.Load() }

// Prepare makes the runtime a page is read through, and is called before a
// window is. Every page this process reads is read through it, and one made
// after a window says nothing about every page it is given.
//
// Nothing is fetched: a machine that holds no runtime says so, and reading is
// what fetches one.
func Prepare(ctx context.Context, cfg Config) error {
	cfg.Download = false
	_, at, err := library(ctx, cfg)
	if err != nil {
		return err
	}
	// A reader is asked for with no models, which is refused after the runtime
	// it would have used is made.
	_, _ = paddle.NewEngine(paddle.Config{OnnxRuntimeLibPath: at})
	standing.Store(true)
	return nil
}
