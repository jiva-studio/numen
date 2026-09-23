package editor

import (
	"context"
	"errors"
	"hash/fnv"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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

func (*asked) Close() error { return nil }

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

// sulking is a model that turns down the first few askings, which is what a
// service that was not answering when the window opened looks like from here.
// What it turned down still owes a vector.
type sulking struct {
	*asked

	mu     sync.Mutex
	left   int
	turned int
}

func newSulking(dims, errs int) *sulking {
	return &sulking{asked: &asked{dims: dims}, left: errs}
}

func (s *sulking) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	s.mu.Lock()
	refusing := s.left > 0
	if refusing {
		s.left--
		s.turned++
	}
	s.mu.Unlock()

	if refusing {
		return nil, errors.New("the model is not answering")
	}
	return s.asked.Embed(ctx, texts)
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
	port.VaultReaders
	walks atomic.Int64
}

func newWalking() *walking { return &walking{VaultReaders: filesystem.VaultReaders{}} }

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

func (r counts) Walk(ctx context.Context, fn func(domain.Fingerprint) error) error {
	r.at.walks.Add(1)
	return r.VaultReader.Walk(ctx, fn)
}

// getVectorProgress is how many of the vault's chunks can carry a vector and how many
// carry one from the model in use.
func getVectorProgress(t *testing.T, f *behind, model port.Embedder) (held, embedded int64) {
	t.Helper()
	held, embedded, err := f.index.Progress().Progress(t.Context(), f.vault.ID, model.Model().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	return held, embedded
}

// TestANoteSavedGetsItsVectorsBack. A save cuts the note again, and its chunks
// owe their vectors from the moment they are written.
func TestANoteSavedGetsItsVectorsBack(t *testing.T) {
	watcher := byHand()
	readers := newWalking()
	model := &asked{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(before, 200),
	}, watcher, readers, model, 20*time.Millisecond)

	waitFor(t, "the note was never embedded at all", func() bool {
		held, embedded := getVectorProgress(t, f, model)
		return held > 0 && held == embedded
	})
	walks := readers.walks.Load()

	write(t, f.vault, "Note.md", noteWith(after, 200))
	sendChange(t, watcher, "Note.md")

	waitFor(t, "the note stayed out of search by meaning", func() bool {
		held, embedded := getVectorProgress(t, f, model)
		return held > 0 && held == embedded && model.saw(after[0])
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
	model := &asked{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(before, 200),
	}, watcher, readers, model, time.Second)

	waitFor(t, "the note was never embedded at all", func() bool {
		held, embedded := getVectorProgress(t, f, model)
		return held > 0 && held == embedded
	})
	first := model.times()

	// Every save leaves the note a word longer, so each one owes a vector the
	// one before it did not.
	for i := range saves {
		write(t, f.vault, "Note.md", noteWith(after, 200+i))
		sendChange(t, watcher, "Note.md")
	}

	waitFor(t, "the saves were never embedded", func() bool {
		held, embedded := getVectorProgress(t, f, model)
		return held > 0 && held == embedded && model.saw(after[0])
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
	if why := f.api.Unwatched.Why(); why == "" {
		t.Fatal("a vault whose watch never started is shown as followed")
	}
	held, embedded := getVectorProgress(t, f, model)
	if held == 0 || embedded > 0 {
		t.Fatalf("%d of %d chunks carry a vector with nothing saved yet", embedded, held)
	}

	// Longer than what is on disk, so every chunk's place is still inside the
	// note the person saved.
	save(t, f, "Note.md", prose(after, 240))

	waitFor(t, "what was saved stayed out of search by meaning", func() bool {
		held, embedded := getVectorProgress(t, f, model)
		return held > 0 && held == embedded && model.saw(after[0])
	})
}

// TestANoteWrittenAfterTheScanFailedIsEmbedded. A vault that could not be read
// is a vault somebody is still writing in, and what they write is embedded.
func TestANoteWrittenAfterTheScanFailedIsEmbedded(t *testing.T) {
	watcher := byHand()
	model := &asked{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(before, 200),
	}, watcher, unwalkable{VaultReaders: filesystem.VaultReaders{}}, model, 20*time.Millisecond)

	waitFor(t, "the scan was not reported as failed", func() bool {
		return f.api.Error.Why() != ""
	})

	write(t, f.vault, "Note.md", noteWith(after, 200))
	sendChange(t, watcher, "Note.md")

	waitFor(t, "what was written stayed out of search by meaning", func() bool {
		held, embedded := getVectorProgress(t, f, model)
		return held > 0 && held == embedded && model.saw(after[0])
	})
}
