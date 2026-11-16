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
	writeFile(t, explicit, []byte("explicit"), 0o644)

	envPath := filepath.Join(dir, "env.onnx")
	writeFile(t, envPath, []byte("env"), 0o644)
	t.Setenv("NUMEN_DEPENDENCY_PP_DOCLAYOUT", envPath)

	dirPath := filepath.Join(dir, "pp-doclayout.onnx")
	writeFile(t, dirPath, []byte("dir"), 0o644)

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

func TestFindInFolderAndHomeFallbacks(t *testing.T) {
	dir := t.TempDir()
	if got := findInFolder("", "locate.go"); got != "" {
		t.Fatalf("findInFolder with empty dir = %q, want empty", got)
	}
	if got := findInFolder(dir, ""); got != "" {
		t.Fatalf("findInFolder with empty fileName = %q, want empty", got)
	}

	if _, err := locateModel(filepath.Join(dir, "missing.onnx"), "", "pp-doclayout.onnx", ""); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("locateModel with missing explicit path = %v, want os.ErrNotExist", err)
	}

	t.Setenv("NUMEN_DEPENDENCY_PP_DOCLAYOUT", filepath.Join(dir, "missing-env.onnx"))
	homeDir := t.TempDir()
	homeModel := filepath.Join(homeDir, "pp-doclayout.onnx")
	writeFile(t, homeModel, []byte("home"), 0o644)
	t.Setenv(EnvDir, homeDir)

	got, err := locateModel("", "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", "")
	if err != nil || got != homeModel {
		t.Fatalf("locateModel via EnvDir = (%q, %v), want (%q, nil)", got, err, homeModel)
	}

	sharedDir := t.TempDir()
	sharedModels := filepath.Join(sharedDir, "models")
	if err := os.MkdirAll(sharedModels, 0o755); err != nil {
		t.Fatal(err)
	}
	sharedModel := filepath.Join(sharedModels, "pp-ocr-det.onnx")
	writeFile(t, sharedModel, []byte("shared"), 0o644)
	t.Setenv("NUMEN_SHARED_DATA_DIR", sharedDir)

	got, err = locateModel("", "", "pp-ocr-det.onnx", "")
	if err != nil || got != sharedModel {
		t.Fatalf("locateModel via shared data dir = (%q, %v), want (%q, nil)", got, err, sharedModel)
	}
}

func writeFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		t.Fatal(err)
	}
}

func TestLocateModelFindsTheFileOfTheEnvironmentPathThatAppearsAfterTheFirstLook(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), "downloaded-det.onnx")
	t.Setenv("NUMEN_DEPENDENCY_PP_OCR_DET", envPath)
	t.Setenv(EnvDir, t.TempDir())
	t.Setenv(cache.EnvSharedDataDir, t.TempDir())

	if found, err := locateModel("", "NUMEN_DEPENDENCY_PP_OCR_DET", "pp-ocr-det.onnx", ""); err == nil {
		t.Fatalf("locateModel found %q before the file was there", found)
	}
	writeFile(t, envPath, []byte("detector"), 0o644)

	found, err := locateModel("", "NUMEN_DEPENDENCY_PP_OCR_DET", "pp-ocr-det.onnx", "")
	if err != nil || found != envPath {
		t.Errorf("locateModel after the download = %q, %v, want %q", found, err, envPath)
	}
}
