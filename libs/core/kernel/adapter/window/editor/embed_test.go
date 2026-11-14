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

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

func getBeforeNames() []string {
	return []string{"river", "mountain", "forest", "meadow", "harbor", "valley", "island", "desert"}
}

func getAfterNames() []string {
	return []string{"lantern", "compass", "anchor", "harvest", "cinder", "willow", "amber", "thistle"}
}

func prose(vocabulary []string, n int) string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, vocabulary[i%len(vocabulary)])
	}
	return strings.Join(out, " ") + "\n"
}

func noteWith(vocabulary []string, n int) string {
	return "---\ntitle: Note\n---\n\n" + prose(vocabulary, n)
}

// fakeEmbedder is a model that returns a vector direction determined by the input
// text. It makes no calls outside this process and counts its requests, which
// gives the number of passes run.
type fakeEmbedder struct {
	dims int

	mu      sync.Mutex
	calls   int
	visited []string
}

func (a *fakeEmbedder) Describe() searchsource.Description {
	return searchsource.Description{Name: "fake", Width: a.dims}
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

// failingEmbedder is a model that rejects the first few requests, simulating a service
// that is unavailable when the window opens. Rejected requests still need a
// vector.
type failingEmbedder struct {
	*fakeEmbedder

	mu            sync.Mutex
	remainingErrs int
	turned        int
}

func newFailingEmbedder(dims, errs int) *failingEmbedder {
	return &failingEmbedder{fakeEmbedder: &fakeEmbedder{dims: dims}, remainingErrs: errs}
}

func (s *failingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
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

func (s *failingEmbedder) getTurnedDown() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.turned
}

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

// walkCountingReaders counts the traversals of a vault. Reading the sources traverses the whole
// vault before embedding; embedding alone opens the files in which the index already
// has a chunk, and does not traverse.
type walkCountingReaders struct {
	vaultport.VaultReaders
	walks atomic.Int64
}

func newWalkCountingReaders() *walkCountingReaders {
	return &walkCountingReaders{VaultReaders: vaultreader.Readers{}}
}

func (w *walkCountingReaders) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := w.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return counts{VaultReader: reader, posVal: w}, nil
}

type counts struct {
	vaultport.VaultReader
	posVal *walkCountingReaders
}

func (r counts) WalkFiles(ctx context.Context, fn func(domain.FileStat) error) error {
	r.posVal.walks.Add(1)
	return r.VaultReader.WalkFiles(ctx, fn)
}

func getVectorProgress(t *testing.T, f *behind, model sourcefake.Embedder) (target, embedded int64) {
	t.Helper()
	target, embedded, err := f.index.Chunks.Progress.GetProgress(t.Context(), f.vault.ID, model.Describe().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	return target, embedded
}

// TestANoteSavedGetsItsVectorsBack. A save cuts the note again, and its chunks
// need their vectors from the moment they are written.
func TestANoteSavedGetsItsVectorsBack(t *testing.T) {
	watcher := newManualWatcher()
	readers := newWalkCountingReaders()
	model := &fakeEmbedder{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(getBeforeNames(), 200),
	}, watcher, readers, model, 20*time.Millisecond)

	waitFor(t, "the note was never embedded at all", func() bool {
		target, embedded := getVectorProgress(t, f, model)
		return target > 0 && target == embedded
	})
	walks := readers.walks.Load()

	write(t, f.vault, "Note.md", noteWith(getAfterNames(), 200))
	sendChange(t, watcher, "Note.md")

	waitFor(t, "the note stayed out of search by meaning", func() bool {
		targetVal, embedded := getVectorProgress(t, f, model)
		return targetVal > 0 && targetVal == embedded && model.saw(getAfterNames()[0])
	})
	if got := readers.walks.Load(); got != walks {
		t.Errorf("the vault was walked %d times over to embed one note", got-walks)
	}
}

// TestTheCooldownDoesNotFirePerSave. The binding in note/tab.ts writes an
// unfinished edit during typing, so saves arrive in quick succession and one
// pass handles them all.
func TestTheCooldownDoesNotFirePerSave(t *testing.T) {
	const saves = 8

	watcher := newManualWatcher()
	readers := newWalkCountingReaders()
	model := &fakeEmbedder{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(getBeforeNames(), 200),
	}, watcher, readers, model, time.Second)

	waitFor(t, "the note was never embedded at all", func() bool {
		target, embedded := getVectorProgress(t, f, model)
		return target > 0 && target == embedded
	})
	first := model.times()

	// Every save lengthens the note by a word, so each one needs a vector the
	// previous save did not.
	for i := range saves {
		write(t, f.vault, "Note.md", noteWith(getAfterNames(), 200+i))
		sendChange(t, watcher, "Note.md")
	}

	waitFor(t, "the saves were never embedded", func() bool {
		targetVal, embedded := getVectorProgress(t, f, model)
		return targetVal > 0 && targetVal == embedded && model.saw(getAfterNames()[0])
	})

	switch passes := model.times() - first; {
	case passes == 0:
		t.Error("nothing asked the model after eight saves")
	case passes > 2:
		t.Errorf("%d saves asked the model %d times", saves, passes)
	}
}

// TestASaveEmbedsWhereTheWatchNeverStarted. A vault with no watcher can still
// be written to, and a save starts the pass that embeds the chunks the index
// lists as missing.
func TestASaveEmbedsWhereTheWatchNeverStarted(t *testing.T) {
	// The first asking is turned down, so the note's chunks come out of the
	// first reading still unembedded their vectors.
	model := newFailingEmbedder(64, 1)

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(getBeforeNames(), 200),
	}, unwatchable{}, newWalkCountingReaders(), model, 20*time.Millisecond)

	waitFor(t, "the model was never asked at all", func() bool { return model.getTurnedDown() > 0 })
	if why := f.api.state.Runtime.GetWatchError(); why == "" {
		t.Fatal("a vault whose watch never started is shown as followed")
	}
	target, embedded := getVectorProgress(t, f, model)
	if target == 0 || embedded > 0 {
		t.Fatalf("%d of %d chunks carry a vector with nothing saved yet", embedded, target)
	}

	// Longer than the file on disk, so every chunk position is within the saved
	// note.
	save(t, f, "Note.md", prose(getAfterNames(), 240))

	waitFor(t, "what was saved stayed out of search by meaning", func() bool {
		targetVal, embedded := getVectorProgress(t, f, model)
		return targetVal > 0 && targetVal == embedded && model.saw(getAfterNames()[0])
	})
}

// TestANoteWrittenAfterTheScanFailedIsEmbedded. A vault that could not be read
// can still be written to, and the notes written are embedded.
func TestANoteSavedAfterTheScanFailedIsEmbedded(t *testing.T) {
	watcher := newManualWatcher()
	model := &fakeEmbedder{dims: 64}

	f := openingWith(t, map[string]string{
		"Note.md": noteWith(getBeforeNames(), 200),
	}, watcher, unwalkable{VaultReaders: vaultreader.Readers{}}, model, 20*time.Millisecond)

	waitFor(t, "the scan was not reported as failed", func() bool {
		return f.api.state.Runtime.GetError() != ""
	})

	write(t, f.vault, "Note.md", noteWith(getAfterNames(), 200))
	sendChange(t, watcher, "Note.md")

	waitFor(t, "what was written stayed out of search by meaning", func() bool {
		target, embedded := getVectorProgress(t, f, model)
		return target > 0 && target == embedded && model.saw(getAfterNames()[0])
	})
}
