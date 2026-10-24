package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocateModelResolvesExplicitPath(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "custom-model.onnx")
	if err := os.WriteFile(explicit, []byte("custom-model"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel(explicit, "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != explicit {
		t.Errorf("got %s, want %s", found, explicit)
	}
}

func TestLocateModelResolvesEnvironmentVariable(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "provisioned-model.onnx")
	if err := os.WriteFile(envPath, []byte("provisioned-model"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NUMEN_DEPENDENCY_PP_DOCLAYOUT", envPath)

	found, err := locateModel("", "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != envPath {
		t.Errorf("got %s, want %s", found, envPath)
	}
}

func TestLocateModelResolvesDirFallback(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "pp-doclayout.onnx")
	if err := os.WriteFile(fallbackPath, []byte("dir-fallback-model"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel("", "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != fallbackPath {
		t.Errorf("got %s, want %s", found, fallbackPath)
	}
}

func TestLocateModelErrorsWhenNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := locateModel("", "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "non-existent.onnx", dir)
	if err == nil {
		t.Fatal("expected error for non-existent model, got nil")
	}
}

func TestLocateModelPrecedence(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "explicit.onnx")
	_ = os.WriteFile(explicit, []byte("explicit"), 0o644)

	envPath := filepath.Join(dir, "env.onnx")
	_ = os.WriteFile(envPath, []byte("env"), 0o644)
	t.Setenv("NUMEN_DEPENDENCY_PP_DOCLAYOUT", envPath)

	dirPath := filepath.Join(dir, "pp-doclayout.onnx")
	_ = os.WriteFile(dirPath, []byte("dir"), 0o644)

	// 1. Explicit beats env and dir
	got, err := locateModel(explicit, "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", dir)
	if err != nil || got != explicit {
		t.Errorf("explicit precedence failed: got %s, err %v", got, err)
	}

	// 2. Env beats dir
	got, err = locateModel("", "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", dir)
	if err != nil || got != envPath {
		t.Errorf("env precedence failed: got %s, err %v", got, err)
	}
}

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
