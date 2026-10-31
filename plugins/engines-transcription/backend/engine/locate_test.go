package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

const (
	encoderVar  = "NUMEN_DEPENDENCY_PARAKEET_ENCODER"
	exampleRepo = "https://example.invalid/parakeet/"
)

func TestLocateModelResolvesAConfiguredPathWithoutARepo(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "custom-encoder.onnx")
	if err := os.WriteFile(explicit, []byte("encoder"), 0o600); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel(explicit, "", encoderVar, encoderFile, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != explicit {
		t.Errorf("got %s, want %s", found, explicit)
	}
}

func TestLocateModelRequiresARepoToSearchTheFolders(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, encoderFile), []byte("encoder"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := locateModel("", "", encoderVar, encoderFile, dir); err == nil {
		t.Fatal("expected an error where no path, variable or repo names the model")
	}
}

func TestLocateModelResolvesTheFolderFallbackOfARepo(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, encoderFile)
	if err := os.WriteFile(fallbackPath, []byte("fallback-encoder"), 0o600); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel("", exampleRepo, encoderVar, encoderFile, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != fallbackPath {
		t.Errorf("got %s, want %s", found, fallbackPath)
	}
}

func TestLocateModelNamesTheFileOfARepoAddress(t *testing.T) {
	dir := t.TempDir()
	named := filepath.Join(dir, "encoder-int8.onnx")
	if err := os.WriteFile(named, []byte("encoder"), 0o600); err != nil {
		t.Fatal(err)
	}

	found, err := locateModel("", exampleRepo+"encoder-int8.onnx", encoderVar, encoderFile, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != named {
		t.Errorf("got %s, want %s", found, named)
	}
}

func TestLocateModelFailsWhenTheRepoFileIsAbsent(t *testing.T) {
	t.Setenv(onnx.ModelsDirVar, t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if _, err := locateModel("", exampleRepo, encoderVar, "non-existent.onnx", t.TempDir()); err == nil {
		t.Fatal("expected an error for a model that exists nowhere")
	}
}
