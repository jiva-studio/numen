package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
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
	if _, err := locateModel(filepath.Join(dir, "missing.onnx"), "", "NUMEN_DEPENDENCY_PARAKEET_ENCODER", encoderFile, dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing explicit path err = %v, want os.ErrNotExist", err)
	}
}

func TestLocateModelEdgeCases(t *testing.T) {
	dir := t.TempDir()
	if got := findInFolder("", "locate.go"); got != "" {
		t.Errorf("findInFolder empty dir = %q, want empty", got)
	}
	if got := findInFolder(dir, ""); got != "" {
		t.Errorf("findInFolder empty fileName = %q, want empty", got)
	}
	if got := resolveFileName(".", "def.onnx"); got != "def.onnx" {
		t.Errorf("resolveFileName(.) = %q, want def.onnx", got)
	}
	if got := resolveFileName("/", "def.onnx"); got != "def.onnx" {
		t.Errorf("resolveFileName(/) = %q, want def.onnx", got)
	}
	if got := resolveFileName("https://example.invalid/repo/custom.onnx", "def.onnx"); got != "custom.onnx" {
		t.Errorf("resolveFileName(custom.onnx) = %q, want custom.onnx", got)
	}

	t.Setenv("NUMEN_DEPENDENCY_PARAKEET_ENCODER", filepath.Join(dir, "missing-env.onnx"))
	if got := findInEnv("NUMEN_DEPENDENCY_PARAKEET_ENCODER"); got != "" {
		t.Errorf("findInEnv missing file = %q, want empty", got)
	}
	if got := findInEnv(""); got != "" {
		t.Errorf("findInEnv empty var = %q, want empty", got)
	}

	homeDir := t.TempDir()
	homeModel := filepath.Join(homeDir, "in-home.onnx")
	writeFile(t, homeModel, []byte("home"), 0o644)
	t.Setenv(EnvDir, homeDir)
	got, err := locateModel("", "https://example.invalid/repo/", "", "in-home.onnx", "")
	if err != nil || got != homeModel {
		t.Errorf("locateModel in home = %q, %v, want %q", got, err, homeModel)
	}

	t.Setenv(EnvDir, t.TempDir())
	sharedData := t.TempDir()
	sharedModels := filepath.Join(sharedData, "models")
	if err := os.MkdirAll(sharedModels, 0o755); err != nil {
		t.Fatal(err)
	}
	sharedModel := filepath.Join(sharedModels, "in-shared.onnx")
	writeFile(t, sharedModel, []byte("shared"), 0o644)
	t.Setenv(cache.EnvSharedDataDir, sharedData)
	if got := findInHome("in-shared.onnx"); got != sharedModel {
		t.Errorf("findInHome shared = %q, want %q", got, sharedModel)
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
	writeFile(t, explicit, []byte("explicit"), 0o644)

	envPath := filepath.Join(dir, "env.onnx")
	writeFile(t, envPath, []byte("env"), 0o644)
	t.Setenv("NUMEN_DEPENDENCY_PARAKEET_ENCODER", envPath)

	dirPath := filepath.Join(dir, encoderFile)
	writeFile(t, dirPath, []byte("dir"), 0o644)

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

func writeFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		t.Fatal(err)
	}
}
