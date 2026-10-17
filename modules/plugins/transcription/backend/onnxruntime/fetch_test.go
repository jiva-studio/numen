package onnxruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
)

func TestCacheDirIsTheFolderTheSettingsName(t *testing.T) {
	got, err := CacheDir(Settings{Dir: "/srv/models"})
	if err != nil || got != "/srv/models" {
		t.Fatalf("CacheDir() = %q, %v", got, err)
	}
}

func TestCacheDirIsTheSharedModelsFolder(t *testing.T) {
	shared := t.TempDir()
	t.Setenv(models.EnvDir, shared)
	got, err := CacheDir(Settings{})
	if err != nil || got != shared {
		t.Fatalf("CacheDir() = %q, %v", got, err)
	}
}

func TestFetchReturnsAFileThatIsAlreadyThere(t *testing.T) {
	dir := t.TempDir()
	address, sum := "https://example.invalid/model.onnx", strings.Repeat("ab", 32)
	at := filepath.Join(dir, models.GetCacheName(address, sum))
	if err := os.WriteFile(at, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Fetch(context.Background(), Settings{Dir: dir}, address, sum)
	if err != nil || got != at {
		t.Fatalf("Fetch() = %q, %v", got, err)
	}
}

func TestFetchNamesTheSectionWhenDownloadsAreOff(t *testing.T) {
	_, err := Fetch(context.Background(), Settings{Dir: t.TempDir(), Section: "indexing.test"}, "https://example.invalid/m.onnx", "")
	if err == nil || !strings.Contains(err.Error(), "indexing.test.download") {
		t.Fatalf("Fetch() error = %v", err)
	}
}
