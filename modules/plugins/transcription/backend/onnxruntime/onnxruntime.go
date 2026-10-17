// Package onnxruntime opens sessions on the ONNX Runtime library the host
// provisioned and gave this process.
package onnxruntime

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	ort "github.com/getcharzp/onnxruntime_purego"
)

const (
	// EnvLibrary names the absolute path of the library the host provisioned.
	EnvLibrary = "NUMEN_ONNX_RUNTIME_LIB"

	// EnvProvider names the execution provider the host chose.
	EnvProvider = "NUMEN_ONNX_PROVIDER"
)

// GetLibrary returns the path of the library the host gave this process.
func GetLibrary() string {
	return os.Getenv(EnvLibrary)
}

// Open returns the engine on the library at path. One engine serves every
// caller, and every library that makes an engine of its own makes it in the same
// breath, so which engine a tensor is built through is settled once and stands
// for the life of the process. An empty path is an error: the host names the
// library for a plugin that declares the runtime.
func Open(path string) (*ort.Engine, string, error) {
	if path == "" {
		return nil, "", fmt.Errorf("no onnx runtime: the host names it in %s", EnvLibrary)
	}
	runtimeState.mu.Lock()
	defer runtimeState.mu.Unlock()
	if runtimeState.engine != nil {
		if runtimeState.at != path {
			return nil, "", fmt.Errorf("the onnx runtime %s is open, and %s is asked for", runtimeState.at, path)
		}
		return runtimeState.engine, runtimeState.at, nil
	}
	preloadCppRuntime()
	engine, err := ort.NewEngine(path)
	if err != nil {
		return nil, "", fmt.Errorf("the onnx runtime %s: %w", path, err)
	}
	recordRuntime(engine, path)
	return engine, path, nil
}

// runtimeState is the runtime this process runs, and where it came from.
//
// One for the life of the process. The lock is over these fields and nothing
// else.
var runtimeState struct {
	mu     sync.Mutex
	engine *ort.Engine
	at     string
	also   []func(at string)
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

// isLoaded says whether this process has its runtime. It is read without the lock.
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

// IsAvailable says whether this process has its runtime, or the file at path is
// there to make one from. It opens nothing.
func IsAvailable(path string) bool {
	if isLoaded.Load() {
		return true
	}
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// HasRegistration says whether any library has registered an initialisation hook.
func HasRegistration() bool {
	runtimeState.mu.Lock()
	defer runtimeState.mu.Unlock()
	return len(runtimeState.also) > 0
}
