// Package onnxruntime is the ONNX Runtime this process runs on: where it is
// found, where it is fetched from, and the engine every model here reads
// through.
package onnxruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// Settings are what the runtime and a model are found by. Section is the
// settings block they were read from, and is what a person is told to change.
type Settings struct {
	Section  string
	Runtime  string
	Dir      string
	Download bool

	// Fetching is told how far a download has got, when anything is listening.
	Fetching func(what string, done, total int64)
}

// say reports how far a download has got, and does nothing when nobody asked.
func (s Settings) say(what string, done, total int64) {
	if s.Fetching != nil {
		s.Fetching(what, done, total)
	}
}

// Open is the ONNX Runtime this machine runs, opened. One engine serves every
// caller here, and every library that makes an engine of its own makes it in the
// same breath, so which engine a tensor is built through is settled once and
// stands for the life of the process.
//
// A path in the settings is used as given. Otherwise what the machine already
// holds is tried, and only a machine holding none fetches one.
//
// One caller opens the runtime and the rest wait on what it is doing, each
// against its own context. Opening it can be a download of many minutes, and a
// caller that gave up on its own is not held to somebody else's.
func Open(ctx context.Context, s Settings) (*ort.Engine, string, error) {
	for {
		held.mu.Lock()
		if held.engine != nil {
			engine, at := held.engine, held.at
			held.mu.Unlock()
			return engine, at, nil
		}
		if stand := held.opening; stand != nil {
			held.mu.Unlock()
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
		held.opening = mine
		held.mu.Unlock()

		engine, at, err := open(ctx, s)

		held.mu.Lock()
		if err == nil {
			keep(engine, at)
		}
		held.opening = nil
		held.mu.Unlock()
		close(mine)
		return engine, at, err
	}
}

// open finds or fetches this machine's runtime. Nothing here is held under the
// lock: it is a download, a copy and two passes of a hash.
func open(ctx context.Context, s Settings) (*ort.Engine, string, error) {
	if s.Runtime != "" {
		engine, err := ort.NewEngine(s.Runtime)
		if err != nil {
			return nil, "", fmt.Errorf("the onnx runtime %s: %w", s.Runtime, err)
		}
		return engine, s.Runtime, nil
	}

	support()
	engine, at, refused := load(candidates(s))
	if engine != nil {
		return engine, at, nil
	}

	found, err := findRelease(s)
	if err != nil {
		return nil, "", err
	}
	archive, err := Fetched(ctx, s, found.address())
	if err != nil {
		return nil, "", fmt.Errorf("the onnx runtime: %w", err)
	}
	at, err = unpack(archive, runtimeName(), found)
	if err != nil {
		return nil, "", err
	}
	if engine, err = ort.NewEngine(at); err != nil {
		refused = append(refused, fmt.Sprintf("%s: %v", at, err))
		return nil, "", fmt.Errorf("no onnx runtime this machine opens — name one in %s.runtime:\n  %s",
			s.Section, strings.Join(refused, "\n  "))
	}
	return engine, at, nil
}

// held is the runtime this process runs, and where it came from.
//
// One for the life of the process. The lock is over these fields and nothing
// else; what one caller is doing to fill them, the rest wait on through opening.
var held struct {
	mu      sync.Mutex
	engine  *ort.Engine
	at      string
	also    []func(at string)
	opening chan struct{}
}

// Alongside is a library that makes an engine of its own, made where this
// process settles its runtime and given the library it opened. It is registered
// from an init, so that it stands before anything opens one.
//
// The binding gives every tensor to the engine made last and stamps it on for
// that tensor's life, so one made on a library's first use moves what everything
// already running was building its tensors through.
func Alongside(also func(at string)) {
	held.mu.Lock()
	defer held.mu.Unlock()
	held.also = append(held.also, also)
}

// here says whether this process has its runtime. It is read without the lock,
// so it is answered while another caller is opening one.
var here atomic.Bool

// keep is the runtime this process has settled on, and the one moment every
// library that makes an engine of its own makes it. The lock is the caller's.
func keep(engine *ort.Engine, at string) {
	held.engine, held.at = engine, at
	for _, also := range held.also {
		also(at)
	}
	here.Store(true)
}

// Here says whether this process has its runtime, or this machine holds a file
// to make one from. It opens nothing, so it is answered while another caller is
// opening one. A name the loader would search for on its own is not a file
// anything here can find.
func Here(s Settings) bool {
	if here.Load() {
		return true
	}
	if s.Runtime != "" {
		_, err := os.Stat(s.Runtime)
		return err == nil
	}
	for _, at := range candidates(s) {
		if filepath.IsAbs(at) {
			return true
		}
	}
	return false
}
