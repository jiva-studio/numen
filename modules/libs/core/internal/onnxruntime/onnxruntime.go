// Package onnxruntime is the ONNX Runtime this process runs on: where it is
// found, where it is fetched from, and the one engine every model reads
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

// Open is the ONNX Runtime this machine runs, opened. One engine serves the
// whole process: the binding builds every tensor through the engine constructed
// last and stamps that engine onto the tensor for its life.
//
// A path in the settings is used as given. Otherwise what the machine already
// holds is tried, and only a machine holding none fetches one.
func Open(ctx context.Context, s Settings) (*ort.Engine, string, error) {
	held.Lock()
	defer held.Unlock()
	if held.engine != nil {
		return held.engine, held.at, nil
	}

	if s.Runtime != "" {
		engine, err := ort.NewEngine(s.Runtime)
		if err != nil {
			return nil, "", fmt.Errorf("the onnx runtime %s: %w", s.Runtime, err)
		}
		return keep(engine, s.Runtime)
	}

	support()
	engine, at, refused := opened(present(s))
	if engine != nil {
		return keep(engine, at)
	}

	found, err := release(s)
	if err != nil {
		return nil, "", err
	}
	archive, err := Fetched(ctx, s, found.address())
	if err != nil {
		return nil, "", fmt.Errorf("the onnx runtime: %w", err)
	}
	at, err = unpacked(archive, runtimeName(), found)
	if err != nil {
		return nil, "", err
	}
	if engine, err = ort.NewEngine(at); err != nil {
		refused = append(refused, fmt.Sprintf("%s: %v", at, err))
		return nil, "", fmt.Errorf("no onnx runtime this machine opens — name one in %s.runtime:\n  %s",
			s.Section, strings.Join(refused, "\n  "))
	}
	return keep(engine, at)
}

// held is the runtime this process runs, and where it came from.
//
// One for the life of the process. The lock is over the opening, which two
// callers may reach at once.
var held struct {
	sync.Mutex
	engine *ort.Engine
	at     string
}

// here says whether this process has its runtime. It is read without the lock,
// so it is answered while another caller is opening one.
var here atomic.Bool

// keep is the runtime this process has settled on. The lock is the caller's.
func keep(engine *ort.Engine, at string) (*ort.Engine, string, error) {
	held.engine, held.at = engine, at
	here.Store(true)
	return engine, at, nil
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
	for _, at := range present(s) {
		if filepath.IsAbs(at) {
			return true
		}
	}
	return false
}
