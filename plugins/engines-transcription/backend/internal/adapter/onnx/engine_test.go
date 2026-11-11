package onnx

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRejectsALibraryThatIsNotThere(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "libonnxruntime.so")
	lib := NewLibrary(func(string) error { return nil })
	if !lib.HasRegistration() || len(lib.runtime.hooks) != 2 {
		t.Fatalf("expected 2 registered hooks on NewLibrary, got %d", len(lib.runtime.hooks))
	}
	if _, _, err := lib.Open(t.Context(), missing); err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("Open() = %v, want %s named", err, missing)
	}
	if _, err := openEngine(missing); err == nil || errors.Unwrap(err) == nil {
		t.Fatalf("openEngine(%q) = %v, want wrapped error", missing, err)
	}
}
