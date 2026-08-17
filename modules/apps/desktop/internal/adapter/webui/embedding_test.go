package webui

import (
	"context"
	"hash/fnv"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// The two vocabularies a note in these tests is written in. A save changes
// which one it holds, so what the model was handed says which body reached it.
var (
	before = []string{"river", "mountain", "forest", "meadow", "harbour", "valley", "island", "desert"}
	after  = []string{"lantern", "compass", "anchor", "harvest", "cinder", "willow", "amber", "thistle"}
)

// noteWith is a note of n words of one vocabulary, long enough to be cut into
// windows that carry a vector.
func noteWith(vocabulary []string, n int) string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, vocabulary[i%len(vocabulary)])
	}
	return "---\ntitle: Note\n---\n\n" + strings.Join(out, " ") + "\n"
}

// asked is a model that answers a direction belonging to the text it was given.
// It reaches nothing outside this process, and counts what it was asked, which
// is how a test says how many passes ran.
type asked struct {
	dims int

	mu    sync.Mutex
	calls int
	seen  []string
}

func (a *asked) Model() port.EmbeddingModel {
	return port.EmbeddingModel{Name: "fake", Dimensions: a.dims}
}

func (a *asked) Embed(_ context.Context, texts []string) ([][]float32, error) {
	a.mu.Lock()
	a.calls++
	a.seen = append(a.seen, texts...)
	a.mu.Unlock()

	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		out = append(out, direction(text, a.dims))
	}
	return out, nil
}

func (a *asked) times() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}

// saw says whether a word reached the model in any text it was handed.
func (a *asked) saw(word string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, text := range a.seen {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
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
	port.VaultReaders
	walks atomic.Int64
}

func walked() *walking { return &walking{VaultReaders: filesystem.Readers{}} }

func (w *walking) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := w.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return counts{VaultReader: reader, at: w}, nil
}

type counts struct {
	port.VaultReader
	at *walking
}

func (r counts) Walk(ctx context.Context, fn func(domain.FileRef) error) error {
	r.at.walks.Add(1)
	return r.VaultReader.Walk(ctx, fn)
}

// vectored is how many of the vault's chunks can carry a vector and how many
// carry one from the model in use.
func vectored(t *testing.T, f *behind, model port.Embedder) (held, embedded int64) {
	t.Helper()
	held, embedded, err := f.index.Progress().Progress(t.Context(), f.vault.ID, model.Model().String())
	if err != nil {
		t.Fatal(err)
	}
	return held, embedded
}

// TestANoteSavedGetsItsVectorsBack. A save cuts the note again, and its chunks
// owe their vectors from the moment they are written.
func TestANoteSavedGetsItsVectorsBack(t *testing.T) {
	watcher := byHand()
	readers := walked()
	model := &asked{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(before, 200),
	}, watcher, readers, model, 20*time.Millisecond)

	eventually(t, "the note was never embedded at all", func() bool {
		held, embedded := vectored(t, f, model)
		return held > 0 && held == embedded
	})
	walks := readers.walks.Load()

	write(t, f.vault, "Note.md", noteWith(after, 200))
	tells(t, watcher, "Note.md")

	eventually(t, "the note stayed out of search by meaning", func() bool {
		held, embedded := vectored(t, f, model)
		return held > 0 && held == embedded && model.saw(after[0])
	})
	if got := readers.walks.Load(); got != walks {
		t.Errorf("the vault was walked %d times over to embed one note", got-walks)
	}
}

// TestTheCooldownDoesNotFirePerSave. The bound in ui/src/tab.ts writes an
// unfinished edit while the person is still typing, so saves arrive one inside
// the next and one pass answers them all.
func TestTheCooldownDoesNotFirePerSave(t *testing.T) {
	const saves = 8

	watcher := byHand()
	readers := walked()
	model := &asked{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(before, 200),
	}, watcher, readers, model, time.Second)

	eventually(t, "the note was never embedded at all", func() bool {
		held, embedded := vectored(t, f, model)
		return held > 0 && held == embedded
	})
	first := model.times()

	// Every save leaves the note a word longer, so each one owes a vector the
	// one before it did not.
	for i := range saves {
		write(t, f.vault, "Note.md", noteWith(after, 200+i))
		tells(t, watcher, "Note.md")
	}

	eventually(t, "the saves were never embedded", func() bool {
		held, embedded := vectored(t, f, model)
		return held > 0 && held == embedded && model.saw(after[0])
	})

	switch passes := model.times() - first; {
	case passes == 0:
		t.Error("nothing asked the model after eight saves")
	case passes > 2:
		t.Errorf("%d saves asked the model %d times", saves, passes)
	}
}
