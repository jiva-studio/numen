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
	for name, want := range map[string]paths{
		"parakeet-tdt-0.6b-v3-int8": {encoder: "parakeet-v3-encoder", decoder: "parakeet-v3-decoder", joiner: "parakeet-v3-joiner", tokens: "parakeet-v3-tokens", segmenter: "silero-vad"},
		"parakeet-tdt-0.6b-v2-int8": {encoder: "parakeet-v2-encoder", decoder: "parakeet-v2-decoder", joiner: "parakeet-v2-joiner", tokens: "parakeet-v2-tokens", segmenter: "silero-vad"},
	} {
		cfg := settings.NewDefaultConfig()
		cfg.Model, _ = settings.FindModel(name)
		var requested []string
		files := modelFiles{dir: t.TempDir(), shouldDownload: true, fetch: func(_ context.Context, id string) (string, error) {
			requested = append(requested, id)
			return "/shared/" + id, nil
		}}
		got, err := files.resolveAll(t.Context(), cfg)
		if err != nil {
			t.Fatal(err)
		}
		if wantPaths := (paths{encoder: "/shared/" + want.encoder, decoder: "/shared/" + want.decoder, joiner: "/shared/" + want.joiner,
			tokens: "/shared/" + want.tokens, segmenter: "/shared/" + want.segmenter}); got != wantPaths {
			t.Errorf("%s: paths = %+v, want %+v", name, got, wantPaths)
		}
		if len(requested) != 5 {
			t.Errorf("%s: requested %v, want the five files of the model", name, requested)
		}
	}
}

func TestALocalModelRequestsNothingFromTheHost(t *testing.T) {
	dir := t.TempDir()
	cfg := settings.NewDefaultConfig()
	for _, name := range []string{encoderFile, decoderFile, joinerFile, tokensFile, "silero.onnx"} {
		writeFile(t, filepath.Join(dir, name), []byte(name), 0o644)
	}
	files := modelFiles{dir: dir, shouldDownload: true, fetch: func(_ context.Context, id string) (string, error) {
		t.Errorf("requested %s for a model in the folder", id)
		return "", errors.New("unexpected request")
	}}
	if _, err := files.resolveAll(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	files.dir = t.TempDir()
	files.shouldDownload = false
	if _, err := files.resolveAll(t.Context(), cfg); err == nil {
		t.Fatal("downloaded an absent model with downloads disabled")
	}
}
