package onnx

import (
	"context"
	"fmt"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// runtimeState is the runtime this process runs, opened once for its life.
var runtimeState Runtime[*ort.Engine]

// Open returns the engine of the library at, opened on the first call. One
// engine serves every caller here, and every library registered makes its own
// engine in the same step.
func Open(ctx context.Context, libPath string) (*ort.Engine, string, error) {
	return runtimeState.Open(ctx, libPath, openEngine)
}

// openEngine loads the library at.
func openEngine(libPath string) (*ort.Engine, error) {
	preloadCppRuntime()
	engine, err := ort.NewEngine(libPath)
	if err != nil {
		return nil, fmt.Errorf("the onnx runtime %s: %w", libPath, err)
	}
	return engine, nil
}

// Register registers a library that creates an engine of its own, created where this
// process initializes its runtime and passed the library it opened. It must be called
// from an init, before anything opens an engine. A hook that returns an error fails the open.
//
// The binding gives every tensor to the engine made last and stamps it on for
// that tensor's life, so one made on a library's first use moves what everything
// already running was building its tensors through.
func Register(also func(libPath string) error) {
	runtimeState.Register(also)
}

// IsAvailable reports whether this process has its runtime, or the library at
// is there to make one from. It opens nothing, so it is answered while another
// caller is opening one.
func IsAvailable(libPath string) bool {
	return runtimeState.IsAvailable(libPath)
}

// HasRegistration reports whether any library has registered an initialisation hook.
func HasRegistration() bool {
	return runtimeState.HasRegistration()
}
