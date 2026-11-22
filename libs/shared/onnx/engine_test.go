package onnx

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRejectsALibraryThatIsNotThere(t *testing.T) {
	lib := NewLibrary()
	if len(lib.runtime.hooks) != 1 {
		t.Fatalf("NewLibrary() has %d hooks, want the graph optimization one", len(lib.runtime.hooks))
	}
	withHooks := NewLibrary(func(string) error { return nil }, func(string) error { return nil })
	if len(withHooks.runtime.hooks) != 3 {
		t.Fatalf("hooks count = %d, want 3", len(withHooks.runtime.hooks))
	}
	missing := filepath.Join(t.TempDir(), "libonnxruntime.so")
	_, err := lib.Open(t.Context(), missing)
	if err == nil || !strings.HasPrefix(err.Error(), "the onnx runtime "+missing) || errors.Unwrap(err) == nil {
		t.Fatalf("Open() = %v, want wrapped error for %s", err, missing)
	}
}
