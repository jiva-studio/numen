package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocateModelResolvesExplicitPath(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "custom-encoder.onnx")
	if err := os.WriteFile(explicit, []byte("encoder"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel(explicit, "", "NUMEN_DEPENDENCY_PARAKEET_ENCODER", encoderFile, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != explicit {
		t.Errorf("got %s, want %s", found, explicit)
	}
}

func TestLocateModelResolvesEnvironmentVariable(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "provisioned-encoder.onnx")
	if err := os.WriteFile(envPath, []byte("encoder"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NUMEN_DEPENDENCY_PARAKEET_ENCODER", envPath)

	found, err := locateModel("", "", "NUMEN_DEPENDENCY_PARAKEET_ENCODER", encoderFile, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != envPath {
		t.Errorf("got %s, want %s", found, envPath)
	}
}

func TestLocateModelResolvesDirFallback(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, encoderFile)
	if err := os.WriteFile(fallbackPath, []byte("fallback-encoder"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel("", "https://example.invalid/parakeet/", "NUMEN_DEPENDENCY_PARAKEET_ENCODER", encoderFile, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != fallbackPath {
		t.Errorf("got %s, want %s", found, fallbackPath)
	}
}

func TestLocateModelErrorsWhenNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := locateModel("", "https://example.invalid/repo/", "NUMEN_DEPENDENCY_PARAKEET_ENCODER", "non-existent.onnx", dir)
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
	t.Setenv("NUMEN_DEPENDENCY_PARAKEET_ENCODER", envPath)

	dirPath := filepath.Join(dir, encoderFile)
	_ = os.WriteFile(dirPath, []byte("dir"), 0o644)

	// 1. Explicit beats env and dir
	got, err := locateModel(explicit, "https://example.invalid/repo/", "NUMEN_DEPENDENCY_PARAKEET_ENCODER", encoderFile, dir)
	if err != nil || got != explicit {
		t.Errorf("explicit precedence failed: got %s, err %v", got, err)
	}

	// 2. Env beats dir
	got, err = locateModel("", "https://example.invalid/repo/", "NUMEN_DEPENDENCY_PARAKEET_ENCODER", encoderFile, dir)
	if err != nil || got != envPath {
		t.Errorf("env precedence failed: got %s, err %v", got, err)
	}
}
