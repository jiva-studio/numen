package onnxruntime

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRefusesALibraryThatIsNotThere(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "libonnxruntime.so")
	if _, _, err := Open(t.Context(), missing); err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("Open() = %v, want %s named", err, missing)
	}
}
