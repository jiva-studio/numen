package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gomlx/go-huggingface/hub"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
)

func TestLocateRefusesToFetchWhileOffline(t *testing.T) {
	t.Setenv(models.EnvOffline, "1")
	t.Setenv(models.EnvDir, t.TempDir())
	_, err := Locate(context.Background(), LocalModel{Name: "owner/repo", ShouldDownload: true}, nil)
	if err == nil || !strings.Contains(err.Error(), models.EnvOffline) {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestLocateRefusesToFetchWhenDownloadIsOff(t *testing.T) {
	_, err := Locate(context.Background(), LocalModel{Name: "owner/repo"}, nil)
	if err == nil || !strings.Contains(err.Error(), "enable download") {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestCheckPinned(t *testing.T) {
	file := filepath.Join(t.TempDir(), "model.onnx")
	if err := os.WriteFile(file, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("weights"))
	sum := hex.EncodeToString(digest[:])

	if err := checkPinned(map[string]string{"onnx/model.onnx": sum}, "onnx/model.onnx", file); err != nil {
		t.Fatalf("matching sum: %v", err)
	}
	err := checkPinned(map[string]string{"onnx/model.onnx": strings.Repeat("0", 64)}, "onnx/model.onnx", file)
	if err == nil || !strings.Contains(err.Error(), "onnx/model.onnx") {
		t.Fatalf("another sum: %v", err)
	}
	if err := checkPinned(map[string]string{"other": sum}, "onnx/model.onnx", file); err != nil {
		t.Fatalf("an unpinned name must not be checked: %v", err)
	}
	if err := checkPinned(nil, "onnx/model.onnx", file); err != nil {
		t.Fatalf("no pins: %v", err)
	}
}

// cacheSnapshot writes a snapshot of owner/repo at revision under the models folder.
func cacheSnapshot(t *testing.T, revision string, files map[string]string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(models.EnvDir, home)
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(home, "huggingface"))
	cache, err := repo.CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		at := filepath.Join(cache, "snapshots", revision, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return cache
}

func sumOf(body string) string {
	digest := sha256.Sum256([]byte(body))
	return hex.EncodeToString(digest[:])
}

func pinnedModel(revision string) LocalModel {
	return LocalModel{
		Name:           "owner/repo",
		ShouldDownload: true,
		Revision:       revision,
		Sums: map[string]string{
			"onnx/model.onnx": sumOf("weights"),
			TokenizerFile:     sumOf("tokens"),
		},
	}
}

func TestLocateUsesAPinnedSnapshotWhileOffline(t *testing.T) {
	cache := cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "tokens"})
	t.Setenv(models.EnvOffline, "1")

	got, err := Locate(context.Background(), pinnedModel("abc"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(cache, "snapshots", "abc", "onnx", "model.onnx"); got.Model != want {
		t.Errorf("model = %q, want %q", got.Model, want)
	}
	if want := filepath.Join(cache, "snapshots", "abc", "tokenizer.json"); got.Tokenizer != want {
		t.Errorf("tokenizer = %q, want %q", got.Tokenizer, want)
	}
}

func TestLocateRefusesACachedModelWithAnotherSum(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "corrupt", "tokenizer.json": "tokens"})
	if _, err := Locate(context.Background(), pinnedModel("abc"), nil); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestLocateRefusesACachedTokenizerWithAnotherSum(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "corrupt"})
	if _, err := Locate(context.Background(), pinnedModel("abc"), nil); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestLocateCachedIgnoresAnIncompleteSnapshot(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "weights"})
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(models.EnvDir), "huggingface"))
	if _, ok, err := locateCached(repo, pinnedModel("abc"), "model.onnx"); ok || err != nil {
		t.Fatalf("locateCached() = %v, %v", ok, err)
	}
}

func TestLocateCachedIgnoresAModelWithNoRevision(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "tokens"})
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(models.EnvDir), "huggingface"))
	if _, ok, err := locateCached(repo, pinnedModel(""), "model.onnx"); ok || err != nil {
		t.Fatalf("locateCached() = %v, %v", ok, err)
	}
}
