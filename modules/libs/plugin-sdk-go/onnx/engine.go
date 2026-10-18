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
func Open(ctx context.Context, at string) (*ort.Engine, string, error) {
	return runtimeState.Open(ctx, at, openEngine)
}

// openEngine loads the library at.
func openEngine(at string) (*ort.Engine, error) {
	preloadCppRuntime()
	engine, err := ort.NewEngine(at)
	if err != nil {
		return nil, fmt.Errorf("the onnx runtime %s: %w", at, err)
	}
	return engine, nil
}

// Register takes a library that makes an engine of its own, made where this
// process settles its runtime and given the library it opened. It is called
// from an init, so that it stands before anything opens one.
//
// The binding gives every tensor to the engine made last and stamps it on for
// that tensor's life, so one made on a library's first use moves what everything
// already running was building its tensors through.
func Register(also func(at string)) {
	runtimeState.Register(also)
}

// IsAvailable reports whether this process has its runtime, or the library at
// is there to make one from. It opens nothing, so it is answered while another
// caller is opening one.
func IsAvailable(at string) bool {
	return runtimeState.IsAvailable(at)
}

// HasRegistration reports whether any library has registered an initialisation hook.
func HasRegistration() bool {
	return runtimeState.HasRegistration()
}
