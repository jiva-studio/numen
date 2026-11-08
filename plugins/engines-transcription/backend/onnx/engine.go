package onnx

import (
	"context"
	"fmt"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// Library is the ONNX Runtime a process runs, opened once for its life, and the calls of it this package resolves. One engine serves every caller.
// The binding gives every tensor to the engine made last and stamps it on for that tensor's life,
// so a library made after others moves what they were already running on.
// A process makes one Library and passes it to everything that runs a model.
type Library struct {
	runtime Runtime[*ort.Engine]
	graph   graphOptimization
	entries configEntryResolver
}

// NewLibrary returns a library that is not open yet. Each hook is called once, with the library
// the engine is opened from, in the same step that records the engine: a library that creates an
// engine of its own makes it there. A hook that returns an error fails the open.
func NewLibrary(hooks ...func(libPath string) error) *Library {
	library := new(Library)
	library.runtime.Register(library.bindGraphOptimization)
	for _, hook := range hooks {
		library.runtime.Register(hook)
	}
	return library
}

// Open returns the engine of the library at libPath, opened on the first call.
func (l *Library) Open(ctx context.Context, libPath string) (*ort.Engine, string, error) {
	return l.runtime.Open(ctx, libPath, openEngine)
}

func openEngine(libPath string) (*ort.Engine, error) {
	// gomutants:disable-next-line STATEMENT_REMOVE reason="C++ runtime preloading is an environment-dependent side effect for Linux dynamic linkers"
	preloadCppRuntime()
	engine, err := ort.NewEngine(libPath)
	if err != nil {
		return nil, fmt.Errorf("the onnx runtime %s: %w", libPath, err)
	}
	return engine, nil
}

// IsAvailable reports whether the library is open, or the library at libPath is there to open. It
// opens nothing, so it is answered while another caller is opening one.
func (l *Library) IsAvailable(libPath string) bool {
	return l.runtime.IsAvailable(libPath)
}

// HasRegistration reports whether any hook runs when the library opens.
func (l *Library) HasRegistration() bool {
	return l.runtime.HasRegistration()
}
