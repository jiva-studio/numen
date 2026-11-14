package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gomlx/go-huggingface/hub"

	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

const (
	testRepo     = "owner/lantern-reranker"
	testRevision = "0123456789abcdef0123456789abcdef01234567"
)

func sumOf(body string) string {
	digest := sha256.Sum256([]byte(body))
	return hex.EncodeToString(digest[:])
}

func pinnedProfile(weights, tokens string) profile.Profile {
	return profile.Profile{
		ID:        testRepo,
		Title:     "Lantern",
		MaxTokens: 512,
		OnnxFile:  "onnx/model.onnx",
		Revision:  testRevision,
		SHA256: map[string]string{
			"onnx/model.onnx": sumOf(weights),
			"tokenizer.json":  sumOf(tokens),
		},
	}
}

func cacheSnapshot(t *testing.T, files map[string]string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(EnvDir, home)
	cache, err := hub.New(testRepo).WithCacheDir(filepath.Join(home, "huggingface")).CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		path := filepath.Join(cache, "snapshots", testRevision, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return cache
}

func TestLocateOfflineWithoutTheModelIsNotOnMachine(t *testing.T) {
	home := t.TempDir()
	t.Setenv(EnvDir, home)
	t.Setenv(EnvOffline, "1")

	_, err := Locate(t.Context(), pinnedProfile("weights", "tokens"), nil)

	if !errors.Is(err, ErrNotOnMachine) {
		t.Fatalf("Locate error = %v, want ErrNotOnMachine", err)
	}
	if !strings.Contains(err.Error(), testRepo) {
		t.Errorf("error %q does not name the model", err)
	}
}

func TestLocateUsesAPinnedSnapshotWhileOffline(t *testing.T) {
	cache := cacheSnapshot(t, map[string]string{"onnx/model.onnx": "weights", "tokenizer.json": "tokens"})
	t.Setenv(EnvOffline, "1")

	got, err := Locate(t.Context(), pinnedProfile("weights", "tokens"), nil)

	if err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(cache, "snapshots", testRevision)
	if got.Model != filepath.Join(snapshot, "onnx", "model.onnx") || got.Tokenizer != filepath.Join(snapshot, "tokenizer.json") {
		t.Errorf("paths = %+v, want the files of the snapshot %s", got, snapshot)
	}
}

func TestLocateDiscardsAFileThatMissesItsPinnedSum(t *testing.T) {
	cache := cacheSnapshot(t, map[string]string{"onnx/model.onnx": "corrupt", "tokenizer.json": "tokens"})
	t.Setenv(EnvOffline, "1")

	_, err := Locate(t.Context(), pinnedProfile("weights", "tokens"), nil)

	if err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("Locate error = %v, want a pinned sum mismatch", err)
	}
	if _, statErr := os.Stat(filepath.Join(cache, "snapshots", testRevision, "onnx", "model.onnx")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("the file that missed its sum stays: %v", statErr)
	}
}

// fakeHub serves one repository as the Hugging Face API does for go-huggingface: the listing of the
// revision and each file by commit. It sends the first half of the model file and holds the rest until
// the test lets it go.
type fakeHub struct {
	files map[string]string
	hold  chan struct{}
}

func (h *fakeHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/models/"+testRepo+"/revision/") {
		h.serveListing(w)
		return
	}
	name, isFile := strings.CutPrefix(r.URL.Path, "/"+testRepo+"/resolve/"+testRevision+"/")
	body, isKnown := h.files[name]
	if !isFile || !isKnown {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("ETag", `"`+sumOf(body)+`"`)
	w.Header().Set("X-Repo-Commit", testRevision)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if r.Method == http.MethodHead {
		return
	}
	if name != "onnx/model.onnx" {
		if _, err := w.Write([]byte(body)); err != nil {
			return
		}
		return
	}
	half := len(body) / 2
	if _, err := w.Write([]byte(body[:half])); err != nil {
		return
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	select {
	case <-h.hold:
	case <-r.Context().Done():
		return
	}
	if _, err := w.Write([]byte(body[half:])); err != nil {
		return
	}
}

func (h *fakeHub) serveListing(w http.ResponseWriter) {
	type sibling struct {
		Name string `json:"rfilename"`
		Size int    `json:"size"`
	}
	listing := struct {
		ID       string    `json:"id"`
		SHA      string    `json:"sha"`
		Siblings []sibling `json:"siblings"`
	}{ID: testRepo, SHA: testRevision}
	for name, body := range h.files {
		listing.Siblings = append(listing.Siblings, sibling{Name: name, Size: len(body)})
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(listing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func TestLocateReportsTheProgressOfTheFirstDownload(t *testing.T) {
	weights := strings.Repeat("lantern weights ", 4096)
	tokens := `{"model":{"type":"Unigram"}}`
	served := &fakeHub{files: map[string]string{"onnx/model.onnx": weights, "tokenizer.json": tokens}, hold: make(chan struct{})}
	server := httptest.NewServer(served)
	t.Cleanup(server.Close)
	t.Setenv("HF_ENDPOINT", server.URL)
	t.Setenv(EnvDir, t.TempDir())
	t.Setenv(EnvOffline, "")
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	var mu sync.Mutex
	var totals []int64
	var release sync.Once
	progress := func(done, total int64) {
		mu.Lock()
		defer mu.Unlock()
		totals = append(totals, total)
		if done < 0 || done > total {
			t.Errorf("progress %d of %d", done, total)
		}
		release.Do(func() { close(served.hold) })
	}

	paths, err := Locate(ctx, pinnedProfile(weights, tokens), progress)

	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if raw, err := os.ReadFile(paths.Model); err != nil || string(raw) != weights {
		t.Errorf("the model file at %s is not the served one: %v", paths.Model, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(totals) == 0 {
		t.Fatal("the download reported no progress")
	}
	if want := int64(len(weights) + len(tokens)); totals[0] != want {
		t.Errorf("progress total = %d, want the %d bytes of the model and the tokenizer", totals[0], want)
	}
}
