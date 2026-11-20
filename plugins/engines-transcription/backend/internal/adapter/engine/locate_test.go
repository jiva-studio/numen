package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
)

func TestLocateModelResolvesExplicitPath(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "custom-encoder.onnx")
	if err := os.WriteFile(explicit, []byte("encoder"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel(explicit, "", encoderFile, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != explicit {
		t.Errorf("got %s, want %s", found, explicit)
	}
	if _, err := locateModel(filepath.Join(dir, "missing.onnx"), "", encoderFile, dir); !errors.Is(err, os.ErrNotExist) {
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

}

func TestLocateModelResolvesDirFallback(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, encoderFile)
	if err := os.WriteFile(fallbackPath, []byte("fallback-encoder"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel("", "https://example.invalid/parakeet/", encoderFile, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != fallbackPath {
		t.Errorf("got %s, want %s", found, fallbackPath)
	}
}

func TestLocateModelErrorsWhenNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := locateModel("", "https://example.invalid/repo/", "non-existent.onnx", dir)
	if err == nil {
		t.Fatal("expected error for non-existent model, got nil")
	}
}

func TestLocateModelPrecedence(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "explicit.onnx")
	writeFile(t, explicit, []byte("explicit"), 0o644)

	dirPath := filepath.Join(dir, encoderFile)
	writeFile(t, dirPath, []byte("dir"), 0o644)

	got, err := locateModel(explicit, "https://example.invalid/repo/", encoderFile, dir)
	if err != nil || got != explicit {
		t.Errorf("explicit precedence failed: got %s, err %v", got, err)
	}

}

func writeFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		t.Fatal(err)
	}
}

func TestResolveSelectedModelThroughHost(t *testing.T) {
	model, _ := settings.FindModel("parakeet-tdt-0.6b-v2-int8")
	address := model.Repo
	ctx := t.Context()
	fetch := func(call context.Context, url, sha string) (string, error) {
		if call != ctx || url != address+encoderFile || sha != "a32b12d17bbbc309d0686fbbcc2987b5e9b8333a7da83fa6b089f0a2acd651ab" {
			t.Fatalf("unexpected model request %s (%s)", url, sha)
		}
		return "/shared/model", nil
	}
	files := modelFiles{dir: t.TempDir(), shouldDownload: true, fetch: fetch}
	got, err := files.resolve(ctx, "", address, encoderFile)
	if err != nil || got != "/shared/model" {
		t.Fatalf("resolve = %q, %v", got, err)
	}
	files.shouldDownload = false
	if _, err := files.resolve(ctx, "", address, encoderFile); err == nil {
		t.Fatal("downloaded an absent model with downloads disabled")
	}
}
