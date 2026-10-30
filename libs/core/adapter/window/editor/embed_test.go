package editor

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// The two vocabularies a note in these tests is written in. A save changes
// which one it holds, so what the model was handed says which body reached it.
var (
	before = []string{"river", "mountain", "forest", "meadow", "harbour", "valley", "island", "desert"}
	after  = []string{"lantern", "compass", "anchor", "harvest", "cinder", "willow", "amber", "thistle"}
)

// prose is n words of one vocabulary, which is what a buffer holds: the text
// under the frontmatter, and no frontmatter of its own.
func prose(vocabulary []string, n int) string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, vocabulary[i%len(vocabulary)])
	}
	return strings.Join(out, " ") + "\n"
}

// noteWith is a note of n words of one vocabulary, long enough to be cut into
// windows that carry a vector.
func noteWith(vocabulary []string, n int) string {
	return "---\ntitle: Note\n---\n\n" + prose(vocabulary, n)
}

// asked is a model that answers a direction belonging to the text it was given.
// It reaches nothing outside this process, and counts what it was asked, which
// is how a test says how many passes ran.
type fakeEmbedder struct {
	dims int

	mu      sync.Mutex
	calls   int
	visited []string
}

func (a *fakeEmbedder) GetModel() embeddingport.EmbeddingModel {
	return embeddingport.EmbeddingModel{Name: "fake", Dimensions: a.dims}
}

func (a *fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	a.mu.Lock()
	a.calls++
	a.visited = append(a.visited, texts...)
	a.mu.Unlock()

	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		out = append(out, direction(text, a.dims))
	}
	return out, nil
}

func (*fakeEmbedder) Close() error { return nil }

func (a *fakeEmbedder) times() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}

// saw says whether a word reached the model in any text it was handed.
func (a *fakeEmbedder) saw(word string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, text := range a.visited {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

// sulking is a model that turns down the first few askings, which is what a
// service that was not answering when the window opened looks like from here.
// What it turned down still owes a vector.
type sulking struct {
	*fakeEmbedder

	mu            sync.Mutex
	remainingErrs int
	turned        int
}

func newSulking(dims, errs int) *sulking {
	return &sulking{fakeEmbedder: &fakeEmbedder{dims: dims}, remainingErrs: errs}
}

func (s *sulking) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	s.mu.Lock()
	refusing := s.remainingErrs > 0
	if refusing {
		s.remainingErrs--
		s.turned++
	}
	s.mu.Unlock()

	if refusing {
		return nil, errors.New("the model is not answering")
	}
	return s.fakeEmbedder.Embed(ctx, texts)
}

// getTurnedDown is how many askings it has turned down.
func (s *sulking) getTurnedDown() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.turned
}

// direction is a vector that belongs to one text and no other.
func direction(text string, dims int) []float32 {
	sum := fnv.New64a()
	sum.Write([]byte(text))
	numbers := rand.New(rand.NewPCG(sum.Sum64(), 0x9E3779B97F4A7C15))

	out := make([]float32, dims)
	for i := range out {
		out[i] = float32(numbers.NormFloat64())
	}
	return out
}

// walking counts the walks of a vault. Reading the sources walks the whole
// vault before it embeds anything; embedding on its own opens the files the
// index already holds a chunk of, and walks nothing.
type walking struct {
	vaultport.VaultReaders
	walks atomic.Int64
}

func newWalking() *walking { return &walking{VaultReaders: vaultreader.Readers{}} }

func (w *walking) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := w.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return counts{VaultReader: reader, posVal: w}, nil
}

type counts struct {
	vaultport.VaultReader
	posVal *walking
}

func (r counts) WalkFiles(ctx context.Context, fn func(domain.FileStat) error) error {
	r.posVal.walks.Add(1)
	return r.VaultReader.WalkFiles(ctx, fn)
}

// getVectorProgress is how many of the vault's chunks can carry a vector and how many
// carry one from the model in use.
func getVectorProgress(t *testing.T, f *behind, model embeddingport.Embedder) (target, embedded int64) {
	t.Helper()
	target, embedded, err := f.index.Progress().GetProgress(t.Context(), f.vault.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	return target, embedded
}

// TestANoteSavedGetsItsVectorsBack. A save cuts the note again, and its chunks
// owe their vectors from the moment they are written.
func TestANoteSavedGetsItsVectorsBack(t *testing.T) {
	watcher := byHand()
	readers := newWalking()
	model := &fakeEmbedder{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(before, 200),
	}, watcher, readers, model, 20*time.Millisecond)

	waitFor(t, "the note was never embedded at all", func() bool {
		target, embedded := getVectorProgress(t, f, model)
		return target > 0 && target == embedded
	})
	walks := readers.walks.Load()

	write(t, f.vault, "Note.md", noteWith(after, 200))
	sendChange(t, watcher, "Note.md")

	waitFor(t, "the note stayed out of search by meaning", func() bool {
		targetVal, embedded := getVectorProgress(t, f, model)
		return targetVal > 0 && targetVal == embedded && model.saw(after[0])
	})
	if got := readers.walks.Load(); got != walks {
		t.Errorf("the vault was walked %d times over to embed one note", got-walks)
	}
}

// TestTheCooldownDoesNotFirePerSave. The bound in note/tab.ts writes an
// unfinished edit while the person is still typing, so saves arrive one inside
// the next and one pass answers them all.
func TestTheCooldownDoesNotFirePerSave(t *testing.T) {
	const saves = 8

	watcher := byHand()
	readers := newWalking()
	model := &fakeEmbedder{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(before, 200),
	}, watcher, readers, model, time.Second)

	waitFor(t, "the note was never embedded at all", func() bool {
		target, embedded := getVectorProgress(t, f, model)
		return target > 0 && target == embedded
	})
	first := model.times()

	// Every save leaves the note a word longer, so each one owes a vector the
	// one before it did not.
	for i := range saves {
		write(t, f.vault, "Note.md", noteWith(after, 200+i))
		sendChange(t, watcher, "Note.md")
	}

	waitFor(t, "the saves were never embedded", func() bool {
		targetVal, embedded := getVectorProgress(t, f, model)
		return targetVal > 0 && targetVal == embedded && model.saw(after[0])
	})

	switch passes := model.times() - first; {
	case passes == 0:
		t.Error("nothing asked the model after eight saves")
	case passes > 2:
		t.Errorf("%d saves asked the model %d times", saves, passes)
	}
}

// TestASaveEmbedsWhereTheWatchNeverStarted. A vault nobody is following is a
// vault somebody is still writing in, and a save is what starts the pass that
// pays what the index says is owed.
func TestASaveEmbedsWhereTheWatchNeverStarted(t *testing.T) {
	// The first asking is turned down, so the note's chunks come out of the
	// first reading still owing their vectors.
	model := newSulking(64, 1)

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(before, 200),
	}, unwatchable{}, newWalking(), model, 20*time.Millisecond)

	waitFor(t, "the model was never asked at all", func() bool { return model.getTurnedDown() > 0 })
	if why := f.api.state.WatchError.Why(); why == "" {
		t.Fatal("a vault whose watch never started is shown as followed")
	}
	target, embedded := getVectorProgress(t, f, model)
	if target == 0 || embedded > 0 {
		t.Fatalf("%d of %d chunks carry a vector with nothing saved yet", embedded, target)
	}

	// Longer than what is on disk, so every chunk's place is still inside the
	// note the person saved.
	save(t, f, "Note.md", prose(after, 240))

	waitFor(t, "what was saved stayed out of search by meaning", func() bool {
		targetVal, embedded := getVectorProgress(t, f, model)
		return targetVal > 0 && targetVal == embedded && model.saw(after[0])
	})
}

// TestANoteWrittenAfterTheScanFailedIsEmbedded. A vault that could not be read
// is a vault somebody is still writing in, and what they write is embedded.
func TestANoteSavedAfterTheScanFailedIsEmbedded(t *testing.T) {
	watcher := byHand()
	model := &fakeEmbedder{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(before, 200),
	}, watcher, unwalkable{VaultReaders: vaultreader.Readers{}}, model, 20*time.Millisecond)

	waitFor(t, "the scan was not reported as failed", func() bool {
		return f.api.state.Error.Why() != ""
	})

	write(t, f.vault, "Note.md", noteWith(after, 200))
	sendChange(t, watcher, "Note.md")

	waitFor(t, "what was written stayed out of search by meaning", func() bool {
		target, embedded := getVectorProgress(t, f, model)
		return target > 0 && target == embedded && model.saw(after[0])
	})
}

func (*fakeEmbedder) Concurrency() int { return 1 }

func (*sulking) Concurrency() int { return 1 }

type mockEmbeddingServer struct{}

func (s *mockEmbeddingServer) GetEmbeddingModel(
	_ context.Context,
	_ *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	return connect.NewResponse(&v1.GetEmbeddingModelResponse{
		Name:       "asked-for",
		Dimensions: 384,
	}), nil
}

func (s *mockEmbeddingServer) EmbedBatch(
	_ context.Context,
	_ *connect.Request[v1.EmbedBatchRequest],
) (*connect.Response[v1.EmbedBatchResponse], error) {
	return connect.NewResponse(&v1.EmbedBatchResponse{}), nil
}

// vaultConfig gives a folder an identity and registers it, and answers with the
// configuration that opens it.
func vaultConfig(t *testing.T, embedding container.EmbeddingConfig) container.Config {
	t.Helper()
	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
		Embedding:    embedding,
	}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vaults.NewAdd(cfg.VaultIdentity(), registry, time.Now).Execute(root, "asked"); err != nil {
		t.Fatal(err)
	}
	return cfg
}

// The embedder an installation configured reaches the search as well as the
// filling of the index, and this asks for it.
func TestTheEmbedderConfiguredIsTheOneOnHand(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.embeddings",
		"name": "Vector Embeddings",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"],
		"backend": {"entrypoint": "bin/embeddings"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(&mockEmbeddingServer{}))
	server := httptest.NewServer(mux)
	defer server.Close()
	serverAddress, ok := server.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("the test server listens on %T, want *net.TCPAddr", server.Listener.Addr())
	}
	serverPort := serverAddress.Port

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(t.Context(), "sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		testsupport.Kill(t, cmd)
		// The plugin host may have reaped the process already.
		if _, err := cmd.Process.Wait(); err != nil && !errors.Is(err, syscall.ECHILD) && !errors.Is(err, os.ErrProcessDone) {
			t.Errorf("wait: %v", err)
		}
	}()

	host.SetLauncher(func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	cfg := vaultConfig(t, container.EmbeddingConfig{})
	cfg.PluginHost = host

	installation, err := Open(
		t.Context(), NewAssembly(t, cfg), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	if installation.Embedder == nil {
		t.Fatal("the vault opened with no embedder on hand")
	}
	if got := installation.Embedder.GetModel().Name; got != "asked-for" {
		t.Errorf("got %q", got)
	}
}

// An installation with no model opens, and says so by having none.
func TestAVaultWithNoModelOpensAnyway(t *testing.T) {
	installation, err := Open(
		t.Context(), NewAssembly(t, vaultConfig(t, container.EmbeddingConfig{})), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	if installation.Embedder != nil {
		t.Errorf("got an embedder: %v", installation.Embedder.GetModel())
	}
}

// Reading a vault is work, and work is one entry in the list of what is being
// done. A pass that reports itself anywhere else draws a second entry for the
// same work, with a count of its own that nobody can place.
func TestAVaultReturnsWhatItIsDoingWhileItReadsItself(t *testing.T) {
	cfg := vaultConfig(t, container.EmbeddingConfig{})

	installation, err := Open(t.Context(), NewAssembly(t, cfg), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	// Said before the reading goroutine starts, so a window that opens on a
	// fresh vault is shown the walk from its first moment.
	list := installation.State.Window.Tasking.List()
	if len(list) != 1 {
		t.Errorf("a vault just opened is doing %+v, want the walk of its notes", list)
	}
	// Nobody asked for it, so it is drawn once it has lasted.
	for _, item := range list {
		if item.IsUserStarted {
			t.Errorf("%q says a person asked for it", item.Title)
		}
	}

	// Each pass takes itself out when it ends, so a vault that has been read
	// says it is doing nothing.
	testsupport.WaitFor(t, func() bool { return len(installation.State.Window.Tasking.List()) == 0 })
}
