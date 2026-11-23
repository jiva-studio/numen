package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
)

func TestLocateModelResolvesExplicitPath(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "custom-model.onnx")
	if err := os.WriteFile(explicit, []byte("custom-model"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel(explicit, "pp-doclayout.onnx", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != explicit {
		t.Errorf("got %s, want %s", found, explicit)
	}
}

func TestLocateModelResolvesDirFallback(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "pp-doclayout.onnx")
	if err := os.WriteFile(fallbackPath, []byte("dir-fallback-model"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel("", "pp-doclayout.onnx", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != fallbackPath {
		t.Errorf("got %s, want %s", found, fallbackPath)
	}
}

func TestLocateModelErrorsWhenNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := locateModel("", "non-existent.onnx", dir)
	if err == nil {
		t.Fatal("expected error for non-existent model, got nil")
	}
}

func TestLocateModelPrecedence(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "explicit.onnx")
	writeFile(t, explicit, []byte("explicit"), 0o644)

	dirPath := filepath.Join(dir, "pp-doclayout.onnx")
	writeFile(t, dirPath, []byte("dir"), 0o644)

	got, err := locateModel(explicit, "pp-doclayout.onnx", dir)
	if err != nil || got != explicit {
		t.Errorf("explicit precedence failed: got %s, err %v", got, err)
	}

}

func TestFindInFolderAndMissingExplicitFile(t *testing.T) {
	dir := t.TempDir()
	if got := findInFolder("", "locate.go"); got != "" {
		t.Fatalf("findInFolder with empty dir = %q, want empty", got)
	}
	if got := findInFolder(dir, ""); got != "" {
		t.Fatalf("findInFolder with empty fileName = %q, want empty", got)
	}

	if _, err := locateModel(filepath.Join(dir, "missing.onnx"), "pp-doclayout.onnx", ""); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("locateModel with missing explicit path = %v, want os.ErrNotExist", err)
	}

}

func writeFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		t.Fatal(err)
	}
}

func TestResolveSelectedModelThroughHost(t *testing.T) {
	address, _ := settings.FindRecognizer("pp-ocrv5-greek")
	ctx := t.Context()
	fetch := func(call context.Context, url, sha string) (string, error) {
		if call != ctx || url != address || sha != "b4368bccd557123c702b7549fee6cd1e94b581337d1c9b65310f109131542b7f" {
			t.Fatalf("unexpected model request %s (%s)", url, sha)
		}
		return "/shared/model", nil
	}
	files := modelFiles{dir: t.TempDir(), shouldDownload: true, fetch: fetch}
	got, err := files.resolve(ctx, "", address, "pp-ocr-rec.onnx")
	if err != nil || got != "/shared/model" {
		t.Fatalf("resolve = %q, %v", got, err)
	}
	files.shouldDownload = false
	if _, err := files.resolve(ctx, "", address, "pp-ocr-rec.onnx"); err == nil {
		t.Fatal("downloaded an absent model with downloads disabled")
	}
}

func TestMissingModelIsNamedWithItsAddress(t *testing.T) {
	address, _ := settings.FindRecognizer("pp-ocrv5-cyrillic")
	refused := errors.New("connection refused")
	fetch := func(context.Context, string, string) (string, error) { return "", refused }
	files := modelFiles{dir: t.TempDir(), shouldDownload: true, fetch: fetch}
	_, err := files.resolve(t.Context(), "", address, "pp-ocr-rec.onnx")
	got := newModelError("recognize", address, "", err)
	if !errors.Is(got, refused) || !strings.Contains(got.Error(), address) || !strings.Contains(got.Error(), "run Recognize again") {
		t.Fatalf("error = %v", got)
	}
	if shown := newModelError("detect", address, "/models/det.onnx", refused).Error(); !strings.Contains(shown, "/models/det.onnx") {
		t.Fatalf("error = %v", shown)
	}
}
