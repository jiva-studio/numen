package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
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

func TestIsFetchedWithConfiguredOrEnvPath(t *testing.T) {
	dir := t.TempDir()
	recPath := filepath.Join(dir, "pp-ocr-rec.onnx")
	if err := os.WriteFile(recPath, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}

	// False when missing
	if IsFetched(settings.Config{}, settings.RecognizerModel{}) {
		t.Fatal("expected IsFetched to be false when nothing configured")
	}

	// True when path is configured
	if !IsFetched(settings.Config{}, settings.RecognizerModel{Path: recPath}) {
		t.Fatal("expected IsFetched to be true with explicit path")
	}

	// True when env var points to existing file
	t.Setenv("NUMEN_DEPENDENCY_PP_OCR_REC", recPath)
	if !IsFetched(settings.Config{}, settings.RecognizerModel{}) {
		t.Fatal("expected IsFetched to be true with env var set")
	}

	// False when explicit path does not exist
	if IsFetched(settings.Config{}, settings.RecognizerModel{Path: filepath.Join(dir, "missing.onnx")}) {
		t.Fatal("expected IsFetched to be false when explicit path is missing")
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

	if _, err := locateModel(filepath.Join(dir, "missing.onnx"), "", "pp-doclayout.onnx", ""); err == nil {
		t.Fatal("locateModel with missing explicit path succeeded, want error")
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
