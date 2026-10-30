package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gomlx/go-huggingface/hub"
)

func TestLocateRefusesToFetchWhileOffline(t *testing.T) {
	t.Setenv(EnvOffline, "1")
	t.Setenv(EnvDir, t.TempDir())
	_, err := Locate(t.Context(), LocalModel{Name: "owner/repo", ShouldDownload: true}, nil)
	if err == nil || !strings.Contains(err.Error(), EnvOffline) {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestLocateRefusesToFetchWhenDownloadIsOff(t *testing.T) {
	_, err := Locate(t.Context(), LocalModel{Name: "owner/repo"}, nil)
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
	missing := filepath.Join(t.TempDir(), "nonexistent")
	if err := checkPinned(map[string]string{"onnx/model.onnx": sum}, "onnx/model.onnx", missing); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("checkPinned error = %v, want fs.ErrNotExist", err)
	}
}

// cacheSnapshot writes a snapshot of owner/repo at revision under the models folder.
func cacheSnapshot(t *testing.T, revision string, files map[string]string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(EnvDir, home)
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(home, "huggingface"))
	cache, err := repo.CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		pos := filepath.Join(cache, "snapshots", revision, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(pos), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pos, []byte(body), 0o644); err != nil {
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
	t.Setenv(EnvOffline, "1")

	got, err := Locate(t.Context(), pinnedModel("abc"), nil)
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
	if _, err := Locate(t.Context(), pinnedModel("abc"), nil); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestLocateRefusesACachedTokenizerWithAnotherSum(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "corrupt"})
	if _, err := Locate(t.Context(), pinnedModel("abc"), nil); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestLocateCachedIgnoresAnIncompleteSnapshot(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "weights"})
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(EnvDir), "huggingface"))
	if _, ok, err := locateCached(repo, pinnedModel("abc"), "model.onnx"); ok || err != nil {
		t.Fatalf("locateCached() = %v, %v", ok, err)
	}
}

func TestLocateCachedIgnoresAModelWithNoRevision(t *testing.T) {
	cache := cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "tokens"})
	rootSnapshot := filepath.Join(cache, "snapshots")
	if err := os.MkdirAll(filepath.Join(rootSnapshot, ModelFolder), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootSnapshot, ModelFolder, "model.onnx"), []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootSnapshot, TokenizerFile), []byte("tokens"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(EnvDir), "huggingface"))
	if _, ok, err := locateCached(repo, pinnedModel(""), "model.onnx"); ok || err != nil {
		t.Fatalf("locateCached() = %v, %v, want false, nil", ok, err)
	}
}

func TestLocateFromDir(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, ModelFile)
	tokenPath := filepath.Join(dir, TokenizerFile)
	if err := os.WriteFile(modelPath, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokenPath, []byte("tokens"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Locate(t.Context(), LocalModel{Dir: dir}, nil)
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if got.Model != modelPath {
		t.Errorf("model = %q, want %q", got.Model, modelPath)
	}
	if got.Tokenizer != tokenPath {
		t.Errorf("tokenizer = %q, want %q", got.Tokenizer, tokenPath)
	}
}

func TestLocateErrorsWhenModelsHomeFails(t *testing.T) {
	t.Setenv(EnvDir, "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("APPDATA", "")
	t.Setenv("LOCALAPPDATA", "")
	_, err := Locate(t.Context(), LocalModel{Name: "owner/repo", ShouldDownload: true}, nil)
	if err == nil || !strings.Contains(err.Error(), "models folder") {
		t.Fatalf("Locate() error = %v, want models folder error", err)
	}
}

func TestLocateCachedFailsWhenCacheDirErrors(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(blocker, "sub"))
	_, _, err := locateCached(repo, LocalModel{Name: "owner/repo", Revision: "rev"}, "model.onnx")
	if err == nil || !strings.Contains(err.Error(), "cache folder of owner/repo") {
		t.Fatalf("locateCached() error = %v, want cache folder error", err)
	}
}

func TestLocateCachedIgnoresMissingModel(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{TokenizerFile: "tokens"})
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(EnvDir), "huggingface"))
	cfg := LocalModel{
		Name:     "owner/repo",
		Revision: "abc",
		Sums:     map[string]string{TokenizerFile: sumOf("tokens")},
	}
	if _, ok, err := locateCached(repo, cfg, "model.onnx"); ok || err != nil {
		t.Fatalf("locateCached() = %v, %v, want false, nil", ok, err)
	}
}

func TestLocateCachedReportsFalseOnCorruptSums(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "corrupt", "tokenizer.json": "tokens"})
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(EnvDir), "huggingface"))
	if _, ok, err := locateCached(repo, pinnedModel("abc"), "model.onnx"); ok || err == nil {
		t.Fatalf("corrupt model: locateCached() = %v, %v, want ok=false and non-nil error", ok, err)
	}

	cacheSnapshot(t, "def", map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "corrupt"})
	repo2 := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(EnvDir), "huggingface"))
	if _, ok, err := locateCached(repo2, pinnedModel("def"), "model.onnx"); ok || err == nil {
		t.Fatalf("corrupt tokenizer: locateCached() = %v, %v, want ok=false and non-nil error", ok, err)
	}
}

func TestLocatePassesRevision(t *testing.T) {
	var mu sync.Mutex
	var requestedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestedPath = r.URL.Path
		mu.Unlock()
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	t.Setenv("HF_ENDPOINT", srv.URL)
	t.Setenv(EnvDir, t.TempDir())
	t.Setenv(EnvOffline, "")

	getPath := func() string {
		mu.Lock()
		defer mu.Unlock()
		return requestedPath
	}
	resetPath := func() {
		mu.Lock()
		defer mu.Unlock()
		requestedPath = ""
	}

	cfg := LocalModel{
		Name:           "owner/repo",
		Revision:       "custom-rev",
		ShouldDownload: true,
	}
	if _, err := Locate(t.Context(), cfg, nil); err == nil {
		t.Fatal("Locate succeeded against a server that answers 404")
	}
	if !strings.Contains(getPath(), "/revision/custom-rev") {
		t.Fatalf("requested path = %q, want revision custom-rev", getPath())
	}

	resetPath()
	cfg.Revision = ""
	if _, err := Locate(t.Context(), cfg, nil); err == nil {
		t.Fatal("Locate succeeded against a server that answers 404")
	}
	if !strings.Contains(getPath(), "/revision/main") {
		t.Fatalf("requested path = %q, want default revision main", getPath())
	}
}
