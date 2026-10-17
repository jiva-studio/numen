// Package onnxruntime is the ONNX Runtime this process runs on: the library the
// host gave it, and the engine every model here reads through.
package onnxruntime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

// Settings are what the runtime is found by: the library a setting names, or else
// the one the host gave this process.
type Settings struct {
	Runtime string
}

// library returns the path of the library to open.
func (s Settings) library() string {
	if s.Runtime != "" {
		return s.Runtime
	}
	return onnx.GetLibrary()
}

// Open is the ONNX Runtime this machine runs, opened. One engine serves every
// caller here, and every library that makes an engine of its own makes it in the
// same breath, so which engine a tensor is built through is settled once and
// stands for the life of the process.
//
// One caller opens the runtime and the rest wait on what it is doing, each
// against its own context.
func Open(ctx context.Context, s Settings) (*ort.Engine, string, error) {
	for {
		runtimeState.mu.Lock()
		if runtimeState.engine != nil {
			engine, at := runtimeState.engine, runtimeState.at
			runtimeState.mu.Unlock()
			return engine, at, nil
		}
		if stand := runtimeState.opening; stand != nil {
			runtimeState.mu.Unlock()
			select {
			case <-stand:
				// Whoever was opening one is done. There is an engine to hand
				// back now, or there is the same nothing to try again.
				continue
			case <-ctx.Done():
				return nil, "", ctx.Err()
			}
		}
		mine := make(chan struct{})
		runtimeState.opening = mine
		runtimeState.mu.Unlock()

		engine, at, err := open(ctx, s)

		runtimeState.mu.Lock()
		if err == nil {
			recordRuntime(engine, at)
		}
		runtimeState.opening = nil
		runtimeState.mu.Unlock()
		close(mine)
		return engine, at, err
	}
}

// open loads the library the settings name. Nothing here is held under the lock.
func open(_ context.Context, s Settings) (*ort.Engine, string, error) {
	if s.library() != "" {
		preloadCppRuntime()
		engine, err := ort.NewEngine(s.library())
		if err != nil {
			return nil, "", fmt.Errorf("the onnx runtime %s: %w", s.library(), err)
		}
		return engine, s.library(), nil
	}
	return nil, "", fmt.Errorf("no onnx runtime: the host names the library of a plugin that declares one")
}

// runtimeState is the runtime this process runs, and where it came from.
//
// One for the life of the process. The lock is over these fields and nothing
// else; what one caller is doing to fill them, the rest wait on through opening.
var runtimeState struct {
	mu      sync.Mutex
	engine  *ort.Engine
	at      string
	also    []func(at string)
	opening chan struct{}
}

// Register takes a library that makes an engine of its own, made where this
// process settles its runtime and given the library it opened. It is called
// from an init, so that it stands before anything opens one.
//
// The binding gives every tensor to the engine made last and stamps it on for
// that tensor's life, so one made on a library's first use moves what everything
// already running was building its tensors through.
func Register(also func(at string)) {
	runtimeState.mu.Lock()
	defer runtimeState.mu.Unlock()
	runtimeState.also = append(runtimeState.also, also)
}

// isLoaded says whether this process has its runtime. It is read without the lock,
// so it is answered while another caller is opening one.
var isLoaded atomic.Bool

// recordRuntime is the runtime this process has settled on, and the one moment every
// library that makes an engine of its own makes it. The lock is the caller's.
func recordRuntime(engine *ort.Engine, at string) {
	runtimeState.engine, runtimeState.at = engine, at
	for _, also := range runtimeState.also {
		also(at)
	}
	isLoaded.Store(true)
}

// IsAvailable says whether this process has its runtime, or the file the settings
// name is there to make one from. It opens nothing, so it is answered while
// another caller is opening one.
func IsAvailable(s Settings) bool {
	if isLoaded.Load() {
		return true
	}
	if s.library() != "" {
		_, err := os.Stat(s.library())
		return err == nil
	}
	return false
}
