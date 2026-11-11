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
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gomlx/go-huggingface/hub"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

func TestLocate_ErrorsWhenOffline(t *testing.T) {
	t.Setenv(EnvOffline, "1")
	t.Setenv(EnvDir, t.TempDir())
	_, err := Locate(t.Context(), model.LocalModel{Name: "owner/repo", ShouldDownload: true}, nil)
	if err == nil || !strings.Contains(err.Error(), EnvOffline) {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestLocate_ErrorsWhenDownloadDisabled(t *testing.T) {
	_, err := Locate(t.Context(), model.LocalModel{Name: "owner/repo"}, nil)
	if err == nil || !strings.Contains(err.Error(), "enable download") {
		t.Fatalf("Locate() error = %v", err)
	}
	if _, err := Locate(t.Context(), model.LocalModel{ShouldDownload: true}, nil); err == nil || !strings.Contains(err.Error(), "no model to run") {
		t.Fatalf("Locate() empty name error = %v", err)
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
	if err := verifyFile(t.TempDir(), sumOf("")); err == nil {
		t.Fatal("verifyFile on a directory returned nil error")
	}
}

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

func pinnedModel(revision string) model.LocalModel {
	return model.LocalModel{
		Name:           "owner/repo",
		ShouldDownload: true,
		Revision:       revision,
		Sums: map[string]string{
			"onnx/model.onnx":   sumOf("weights"),
			model.TokenizerFile: sumOf("tokens"),
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

func TestLocate_ErrorsOnModelChecksumMismatch(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "corrupt", "tokenizer.json": "tokens"})
	if _, err := Locate(t.Context(), pinnedModel("abc"), nil); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestLocate_ErrorsOnTokenizerChecksumMismatch(t *testing.T) {
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

func TestLocateCachedReturnsTheSnapshotOfAModelWithNoRevision(t *testing.T) {
	cache := cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "tokens"})
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(EnvDir), "huggingface"))

	got, ok, err := locateCached(repo, pinnedModel(""), "model.onnx")
	if err != nil || !ok {
		t.Fatalf("locateCached() = %v, %v, want true, nil", ok, err)
	}
	if want := filepath.Join(cache, "snapshots", "abc", "onnx", "model.onnx"); got.Model != want {
		t.Errorf("model = %q, want %q", got.Model, want)
	}
}

func TestLocateCachedReturnsNewestSnapshotWithoutRevision(t *testing.T) {
	cache := cacheSnapshot(t, "old", map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "tokens"})
	for name, body := range map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "tokens"} {
		pos := filepath.Join(cache, "snapshots", "new", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(pos), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pos, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	older := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(cache, "snapshots", "old", "onnx", "model.onnx"), older, older); err != nil {
		t.Fatal(err)
	}
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(EnvDir), "huggingface"))

	got, ok, err := locateCached(repo, pinnedModel(""), "model.onnx")
	if err != nil || !ok {
		t.Fatalf("locateCached() = %v, %v, want true, nil", ok, err)
	}
	if want := filepath.Join(cache, "snapshots", "new", "onnx", "model.onnx"); got.Model != want {
		t.Errorf("model = %q, want %q", got.Model, want)
	}
}

func TestLocateCachedSkipsIncompleteSnapshotWithoutRevision(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "weights"})
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(EnvDir), "huggingface"))
	if _, ok, err := locateCached(repo, pinnedModel(""), "model.onnx"); ok || err != nil {
		t.Fatalf("locateCached() = %v, %v, want false, nil", ok, err)
	}
}

func TestLocateCachedFindsNothingWhereTheCacheHasNoSnapshots(t *testing.T) {
	t.Setenv(EnvDir, t.TempDir())
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(EnvDir), "huggingface"))
	if _, ok, err := locateCached(repo, pinnedModel(""), "model.onnx"); ok || err != nil {
		t.Fatalf("locateCached() = %v, %v, want false, nil", ok, err)
	}
}

func TestLocateUsesCachedSnapshotWithoutRevisionOffline(t *testing.T) {
	cache := cacheSnapshot(t, "abc", map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "tokens"})
	t.Setenv(EnvOffline, "1")

	got, err := Locate(t.Context(), model.LocalModel{Name: "owner/repo", ShouldDownload: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(cache, "snapshots", "abc", "tokenizer.json"); got.Tokenizer != want {
		t.Errorf("tokenizer = %q, want %q", got.Tokenizer, want)
	}
}

func TestLocateFromDir(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, model.ModelFile)
	tokenPath := filepath.Join(dir, model.TokenizerFile)
	if err := os.WriteFile(modelPath, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Locate(t.Context(), model.LocalModel{Dir: dir}, nil); err == nil {
		t.Fatal("Locate() succeeded with missing tokenizer")
	}
	if err := os.WriteFile(tokenPath, []byte("tokens"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Locate(t.Context(), model.LocalModel{Dir: dir}, nil)
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
	t.Setenv(cache.EnvSharedDataDir, "")
	t.Setenv(cache.EnvDataDir, "")
	_, err := Locate(t.Context(), model.LocalModel{Name: "owner/repo", ShouldDownload: true}, nil)
	if err == nil || !strings.Contains(err.Error(), "models folder") || errors.Unwrap(err) == nil {
		t.Fatalf("Locate() error = %v, want wrapped models folder error", err)
	}
}

func TestLocateCachedFailsWhenCacheDirErrors(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(blocker, "sub"))
	_, ok, err := locateCached(repo, model.LocalModel{Name: "owner/repo", Revision: "rev"}, "model.onnx")
	if ok || err == nil || !strings.Contains(err.Error(), "cache folder of owner/repo") || errors.Unwrap(err) == nil {
		t.Fatalf("locateCached() = %v, %v, want false and wrapped cache folder error", ok, err)
	}
}

func TestLocateCachedIgnoresMissingModel(t *testing.T) {
	cacheSnapshot(t, "abc", map[string]string{model.TokenizerFile: "tokens"})
	repo := hub.New("owner/repo").WithCacheDir(filepath.Join(os.Getenv(EnvDir), "huggingface"))
	cfg := model.LocalModel{
		Name:     "owner/repo",
		Revision: "abc",
		Sums:     map[string]string{model.TokenizerFile: sumOf("tokens")},
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

	cfg := model.LocalModel{
		Name:           "owner/repo",
		Revision:       "custom-rev",
		ShouldDownload: true,
	}
	if _, err := Locate(t.Context(), cfg, nil); err == nil || !strings.Contains(err.Error(), "listing owner/repo:") || errors.Unwrap(err) == nil {
		t.Fatalf("Locate against 404 = %v, want wrapped listing error", err)
	}
	if !strings.Contains(getPath(), "/revision/custom-rev") {
		t.Fatalf("requested path = %q, want revision custom-rev", getPath())
	}

	repo := hub.New("owner/repo").WithProgressBar(false).WithCacheDir(filepath.Join(t.TempDir(), "hf"))
	if _, _, tokSize, err := listRepoFiles(repo); err == nil || tokSize != 0 {
		t.Fatalf("listRepoFiles against 404 = %d, %v", tokSize, err)
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

func TestAFileThatMissesItsPinnedSumIsRemovedWithItsBlobAndFetchedAgain(t *testing.T) {
	dir := t.TempDir()
	blob := filepath.Join(dir, "blobs", "etag")
	link := filepath.Join(dir, "snapshots", "rev", "tokenizer.json")
	for _, folder := range []string{filepath.Dir(blob), filepath.Dir(link)} {
		if err := os.MkdirAll(folder, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(blob, []byte("truncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(blob, link); err != nil {
		t.Skipf("symlinks are not available: %v", err)
	}

	err := checkPinned(map[string]string{model.TokenizerFile: strings.Repeat("0", 64)}, model.TokenizerFile, link)

	if err == nil {
		t.Fatal("checkPinned() accepted a file that holds another sum")
	}
	for _, path := range []string{blob, link} {
		if _, statErr := os.Lstat(path); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("%s stays after the file missed its sum: %v", path, statErr)
		}
	}
}

func TestSelectModelFilesSelectsTheDataFileOfTheNamedModelOnly(t *testing.T) {
	folder := []string{
		"model.onnx", "model.onnx_data", "model_fp16.onnx", "model_fp16.onnx_data", "model_q4.onnx", "model_q4.onnx_data",
		"model_quantized.onnx", "model_quantized.onnx_data",
	}
	got := selectModelFiles(folder, "model_quantized.onnx")
	want := []string{"model_quantized.onnx", "model_quantized.onnx_data"}
	if !slices.Equal(got, want) {
		t.Errorf("selectModelFiles = %v, want %v", got, want)
	}
}

func TestDiscardDownloadAndSnapshotEdgeCases(t *testing.T) {
	if err := discardDownload(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Errorf("discardDownload(missing) = %v, want nil", err)
	}
	regular := filepath.Join(t.TempDir(), "regular")
	if err := os.WriteFile(regular, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := discardDownload(regular); err != nil {
		t.Errorf("discardDownload(regular) = %v, want nil", err)
	}
	nonEmptyDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "child"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = discardDownload(nonEmptyDir)
	if err == nil {
		t.Fatal("discardDownload(nonEmptyDir) = nil, want error")
	}
	if joined, ok := err.(interface{ Unwrap() []error }); !ok || len(joined.Unwrap()) != 1 {
		t.Errorf("discardDownload(nonEmptyDir) unwrapped errors = %v, want 1 error", err)
	}

	cacheDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(cacheDir, "snapshots"), []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := locateNewestSnapshot(cacheDir, "owner/repo", "model.onnx"); ok || err == nil {
		t.Errorf("locateNewestSnapshot with file snapshots = %v, %v", ok, err)
	}

	snapRoot := t.TempDir()
	// Snapshot "a-incomplete" has tokenizer.json as a file and onnx/model.onnx as a directory.
	if err := os.MkdirAll(filepath.Join(snapRoot, "snapshots", "a-incomplete", "onnx", "model.onnx"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snapRoot, "snapshots", "a-incomplete", model.TokenizerFile), []byte("t"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Snapshot "b-complete" has both files with zero Time.
	bModel := filepath.Join(snapRoot, "snapshots", "b-complete", "onnx", "model.onnx")
	if err := os.MkdirAll(filepath.Dir(bModel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bModel, []byte("m"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snapRoot, "snapshots", "b-complete", model.TokenizerFile), []byte("t"), 0o644); err != nil {
		t.Fatal(err)
	}
	zeroTime := time.Time{}
	if err := os.Chtimes(bModel, zeroTime, zeroTime); err != nil {
		t.Fatal(err)
	}

	got, ok, err := locateNewestSnapshot(snapRoot, "owner/repo", "model.onnx")
	if err != nil || !ok || got.Model != bModel {
		t.Errorf("locateNewestSnapshot = %+v, %v, %v, want %q", got, ok, err, bModel)
	}
}
