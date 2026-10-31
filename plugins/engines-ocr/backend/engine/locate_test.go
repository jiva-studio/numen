package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFetchedWithConfiguredOrEnvPath(t *testing.T) {
	dir := t.TempDir()
	recPath := filepath.Join(dir, "pp-ocr-rec.onnx")
	if err := os.WriteFile(recPath, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}

	// False when missing
	if IsFetched(Config{}, RecogniserModel{}) {
		t.Fatal("expected IsFetched to be false when nothing configured")
	}

	// True when path is configured
	if !IsFetched(Config{}, RecogniserModel{Path: recPath}) {
		t.Fatal("expected IsFetched to be true with explicit path")
	}

	// True when env var points to existing file
	t.Setenv("NUMEN_DEPENDENCY_PP_OCR_REC", recPath)
	if !IsFetched(Config{}, RecogniserModel{}) {
		t.Fatal("expected IsFetched to be true with env var set")
	}
}
