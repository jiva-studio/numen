package onnx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
)

// ErrNoLibrary is the error of opening a runtime with no library named.
var ErrNoLibrary = errors.New("no onnx runtime: the host names the library of a plugin that declares one")

// Runtime is the one ONNX Runtime a process opens, as the engine E its binding
// makes. The zero value is ready to use.
//
// One caller opens the runtime and the rest wait on it, each against its own
// context. The engine and its library stand for the life of the process.
type Runtime[E any] struct {
	mu      sync.Mutex
	engine  E
	libPath string
	hooks   []func(libPath string) error
	opening chan struct{}
	loaded  atomic.Bool
}

// Open returns the engine of this process, made from the library at by load
// the first time it succeeds. Every hook registered runs once, with the
// library, in the same step that records the engine. A hook that fails fails
// the open with the error of every failing hook, and nothing is recorded.
func (r *Runtime[E]) Open(ctx context.Context, libPath string, load func(libPath string) (E, error)) (E, string, error) {
	var none E
	for {
		r.mu.Lock()
		if r.loaded.Load() {
			engine, openedPath := r.engine, r.libPath
			r.mu.Unlock()
			return engine, openedPath, nil
		}
		if pending := r.opening; pending != nil {
			r.mu.Unlock()
			select {
			case <-pending:
				continue
			case <-ctx.Done():
				return none, "", ctx.Err()
			}
		}
		mine := make(chan struct{})
		r.opening = mine
		r.mu.Unlock()

		engine, err := r.load(libPath, load)

		r.mu.Lock()
		if err == nil {
			err = r.record(engine, libPath)
		}
		r.opening = nil
		r.mu.Unlock()
		close(mine)
		if err != nil {
			return none, "", err
		}
		return engine, libPath, nil
	}
}

// load makes the engine. Nothing here is held under the lock.
func (r *Runtime[E]) load(libPath string, load func(libPath string) (E, error)) (E, error) {
	if libPath == "" {
		var none E
		return none, ErrNoLibrary
	}
	return load(libPath)
}

// record runs every hook and keeps the engine when none fails. The lock is the caller's.
func (r *Runtime[E]) record(engine E, libPath string) error {
	var failures []error
	for _, hook := range r.hooks {
		if err := hook(libPath); err != nil {
			failures = append(failures, err)
		}
	}
	if err := errors.Join(failures...); err != nil {
		return fmt.Errorf("bind the onnx runtime %s: %w", libPath, err)
	}
	r.engine, r.libPath = engine, libPath
	r.loaded.Store(true)
	return nil
}

// Register adds a hook that runs once with the library the runtime is opened
// from. A hook registered after the runtime opened never runs.
//
// A hook runs under the runtime's lock and works only with the path it is
// given; it calls no method of this Runtime.
func (r *Runtime[E]) Register(hook func(libPath string) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, hook)
}

// HasRegistration reports whether any hook is registered.
func (r *Runtime[E]) HasRegistration() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.hooks) > 0
}

// GetLoaded returns the engine and its library, or the zero engine and an empty
// path while the runtime is not open.
func (r *Runtime[E]) GetLoaded() (E, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.engine, r.libPath
}

// IsAvailable reports whether the runtime is open, or the library at is there
// to open it from. It opens nothing and does not wait on a caller that is
// opening it.
func (r *Runtime[E]) IsAvailable(libPath string) bool {
	if r.loaded.Load() {
		return true
	}
	if libPath == "" {
		return false
	}
	_, err := os.Stat(libPath)
	return err == nil
}
