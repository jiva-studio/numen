package onnx

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
)

func TestOpenRefusesALibraryThatIsNotThere(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "libonnxruntime.so")
	if _, _, err := Open(t.Context(), missing); err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("Open() = %v, want %s named", err, missing)
	}
}

func TestOpenAndLocateRefusesAMissingLibrary(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "libonnxruntime.so")
	var got string
	_, _, err := OpenAndLocate(t.Context(), missing, models.Folder{}, nil, []models.Wanted{
		{Dst: &got, Path: "model.onnx", Kind: "test model"},
	})
	if err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("OpenAndLocate() = %v, want %s named", err, missing)
	}
}

