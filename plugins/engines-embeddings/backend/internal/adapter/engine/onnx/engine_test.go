package onnx

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRejectsALibraryThatIsNotThere(t *testing.T) {
	if (new(Library)).HasRegistration() {
		t.Fatal("zero Library has registration before NewLibrary")
	}
	lib := NewLibrary()
	if !lib.HasRegistration() {
		t.Fatal("NewLibrary() has no default registration")
	}
	withHooks := NewLibrary(func(string) error { return nil }, func(string) error { return nil })
	if len(withHooks.runtime.hooks) != 3 {
		t.Fatalf("hooks count = %d, want 3", len(withHooks.runtime.hooks))
	}
	missing := filepath.Join(t.TempDir(), "libonnxruntime.so")
	_, _, err := lib.Open(t.Context(), missing)
	if err == nil || !strings.HasPrefix(err.Error(), "the onnx runtime "+missing) || errors.Unwrap(err) == nil {
		t.Fatalf("Open() = %v, want wrapped error for %s", err, missing)
	}
}
