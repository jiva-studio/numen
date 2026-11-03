package vectors

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

// dimensions is the width of the model the tests embed with. One byte per
// dimension and one bit per dimension are two lengths, so a representation
// written in the place of the other is visible.
const dimensions = 32

// cutBooks extracts one vault's books and answers with the chunks that need a vector.
func cutBooks(t testing.TB, index *store, shelf *library, v domain.Vault) []storedChunk {
	t.Helper()
	if _, err := (newExtract(vaults{v.ID: shelf}, index, index)).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	small := index.small(v.ID)
	if len(small) == 0 {
		t.Fatal("the book was not cut into anything that carries a vector")
	}
	return small
}

func TestEmbeddingCarriesOnWhereItStopped(t *testing.T) {
	// The book is long enough that the chunks still needing a vector are requested more than
	// once, so the query pages as well as resumes.
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindFile,
		bookOf(t, "A Long Book", words(getSanskrit(), 4500), words(getSanskrit(), 4500)), 1)

	small := cutBooks(t, index, shelf, getFirst())
	if len(small) <= chunksPerQuery {
		t.Fatalf("%d chunks need a vector, which one question answers with: the run would never resume", len(small))
	}

	stopCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	interrupted := &interruptedStore{store: index, cancel: cancel, after: 2}
	embed := newEmbed(vaults{getFirst().ID: shelf}, index, interrupted)
	embed.Embedder = &embedder{dims: dimensions}
	embed.BatchCharacters = 4000
	if _, err := embed.Execute(stopCtx, getFirst()); err == nil {
		t.Fatal("a cancelled context did not stop the run")
	}

	part := len(index.vectors)
	if interrupted.saved != interrupted.after || part == 0 || part == len(small) {
		t.Fatalf("the interrupted run embedded %d of %d chunks, want some of them", part, len(small))
	}
	// A chunk is written before its vector, so every vector the interrupted run
	// left names a chunk that is there.
	for chunk := range index.vectors {
		if !index.hasChunk(chunk) {
			t.Fatalf("a vector was written for chunk %s, which the index does not hold", chunk)
		}
	}

	embed.vectors = index
	res, err := embed.Execute(ctx, getFirst())
	if err != nil {
		t.Fatal(err)
	}

	// The two halves of resumption: nothing was embedded twice, and nothing was
	// skipped.
	if want := len(small) - part; res.Embedded != want {
		t.Errorf("the second run embedded %d chunks, want the %d the first did not", res.Embedded, want)
	}
	for _, chunk := range small {
		switch len(index.getVectors(chunk)) {
		case 1:
		case 0:
			t.Fatalf("chunk %d was skipped by both runs", chunk.id)
		default:
			t.Fatalf("chunk %d was embedded %d times", chunk.id, len(index.getVectors(chunk)))
		}
	}
	if len(index.vectors) != len(small) {
		t.Errorf("%d chunks carry a vector, want the %d that unembedded one", len(index.vectors), len(small))
	}

	// No chunk needs a vector, so a third run does no work.
	third, err := embed.Execute(ctx, getFirst())
	if err != nil {
		t.Fatal(err)
	}
	if third.Pending != 0 || third.Embedded != 0 {
		t.Errorf("a run after everything was embedded reports %+v", third)
	}
}

func TestBothRepresentationsOfAVectorAreSavedTogether(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindFile, bookOf(t, "A Book", words(getSanskrit(), 400), words(getSanskrit(), 400)), 1)
	small := cutBooks(t, index, shelf, getFirst())

	var progress []EmbedResult
	model := &embedder{dims: dimensions}
	emb := newEmbed(vaults{getFirst().ID: shelf}, index, index)
	emb.Embedder = model
	emb.BatchCharacters = 2000
	emb.OnProgress = func(r EmbedResult) { progress = append(progress, r) }
	res, err := emb.Execute(ctx, getFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Embedded != len(small) {
		t.Errorf("embedded = %d, want the %d chunks that unembedded a vector", res.Embedded, len(small))
	}

	if len(index.groups) < 2 {
		t.Fatalf("the vectors arrived in %d writes, want a group at a time", len(index.groups))
	}
	for _, group := range index.groups {
		for _, v := range group {
			if v.Kind != embeddingport.QuantisedInt8 {
				t.Errorf("chunk %s was stored as %q", v.ChunkID, v.Kind)
			}
			if len(v.Value) != dimensions {
				t.Errorf("chunk %s holds %d bytes for %d dimensions", v.ChunkID, len(v.Value), dimensions)
			}
			if want := (dimensions + 7) / 8; len(v.Coarse) != want {
				t.Errorf("chunk %s holds %d coarse bytes, want %d", v.ChunkID, len(v.Coarse), want)
			}
			if v.Model != model.GetModel() {
				t.Errorf("chunk %s was stored under %s", v.ChunkID, v.Model)
			}
		}
	}
	for _, chunk := range small {
		if len(index.getVectors(chunk)) != 1 {
			t.Fatalf("chunk %d carries %d vectors", chunk.id, len(index.getVectors(chunk)))
		}
	}

	if len(progress) < len(index.groups) {
		t.Errorf("progress was reported %d times for %d groups", len(progress), len(index.groups))
	}
	if last := progress[len(progress)-1]; last.CurrentSource != bookPath {
		t.Errorf("progress says it is extracting %q", last.CurrentSource)
	}
}

func TestWithNoEmbedderTheTextIsSplitAndNothingFails(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindFile, bookOf(t, "A Book", words(getSanskrit(), 400)), 1)
	small := cutBooks(t, index, shelf, getFirst())

	res, err := (newEmbed(vaults{getFirst().ID: shelf}, index, index)).Execute(ctx, getFirst())
	if err != nil {
		t.Fatalf("a vault with no embedder is not an error: %v", err)
	}
	if !reflect.DeepEqual(res, EmbedResult{}) {
		t.Errorf("the run reports %+v, want nothing done", res)
	}
	if len(index.vectors) != 0 || len(index.groups) != 0 {
		t.Errorf("%d chunks carry a vector with no model to make one", len(index.vectors))
	}

	// What the words half answers over is there all the same.
	for _, chunk := range small {
		if chunk.text == "" {
			t.Fatalf("chunk %d holds no text for a search by words", chunk.id)
		}
	}
}

func TestAChunkWhoseSourceMovedOnIsLeftAsItIs(t *testing.T) {
	cases := []struct {
		name    string
		replace func(t *testing.T, shelf *library)
		want    func(t *testing.T, res EmbedResult, unembedded int)
	}{
		{
			name:    "the file is gone",
			replace: func(_ *testing.T, shelf *library) { delete(shelf.files, bookPath) },
			want: func(t *testing.T, res EmbedResult, unembedded int) {
				if res.Vanished != unembedded || res.Embedded != 0 {
					t.Errorf("the run reports %+v, want all %d chunks left alone", res, unembedded)
				}
			},
		},
		{
			name: "the file is no longer a book",
			replace: func(_ *testing.T, shelf *library) {
				shelf.hold(bookPath, domain.KindFile, []byte("not an archive"), 2)
			},
			want: func(t *testing.T, res EmbedResult, unembedded int) {
				if res.Vanished != unembedded || res.Embedded != 0 {
					t.Errorf("the run reports %+v, want all %d chunks left alone", res, unembedded)
				}
			},
		},
		{
			name: "the file holds another text now, and the version the chunks were cut from is gone",
			replace: func(t *testing.T, shelf *library) {
				shelf.hold(bookPath, domain.KindFile, bookOf(t, "A Book", words(getSanskrit(), 20)), 2)
			},
			want: func(t *testing.T, res EmbedResult, unembedded int) {
				if res.Vanished != unembedded || res.Embedded != 0 {
					t.Errorf("the run reports %+v, want all %d chunks left alone", res, unembedded)
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := t.Context()
			index, shelf := newStore(), newLibrary()
			shelf.hold(bookPath, domain.KindFile, bookOf(t, "A Book", words(getSanskrit(), 400)), 1)
			small := cutBooks(t, index, shelf, getFirst())
			c.replace(t, shelf)

			emb := newEmbed(vaults{getFirst().ID: shelf}, index, index)
			emb.Embedder = &embedder{dims: dimensions}
			res, err := emb.Execute(ctx, getFirst())
			if err != nil {
				t.Fatalf("a source that moved on is not an error: %v", err)
			}
			if res.Pending != len(small) {
				t.Errorf("unembedded = %d, want the %d chunks with no vector", res.Pending, len(small))
			}
			if res.Embedded+res.Vanished+res.Displaced != res.Pending {
				t.Errorf("the run reports %+v, and every chunk it was given is embedded or accounted for", res)
			}
			c.want(t, res, len(small))
		})
	}
}

func TestEmbeddingStaysInsideItsVault(t *testing.T) {
	ctx := t.Context()
	index := newStore()
	shelves := vaults{getFirst().ID: newLibrary(), getSecond().ID: newLibrary()}
	shelves[getFirst().ID].hold("library/sanskrit.epub", domain.KindFile,
		bookOf(t, "Sanskrit", words(getSanskrit(), 400)), 1)
	shelves[getSecond().ID].hold("library/latin.epub", domain.KindFile,
		bookOf(t, "Latin", words(getLatin(), 400)), 1)

	extract := newExtract(shelves, index, index)
	for _, v := range []domain.Vault{getFirst(), getSecond()} {
		if _, err := extract.Execute(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	mine, theirs := index.small(getFirst().ID), index.small(getSecond().ID)
	if len(mine) == 0 || len(theirs) == 0 {
		t.Fatal("both vaults have to hold chunks for this to say anything")
	}

	embed := newEmbed(shelves, index, index)
	embed.Embedder = &embedder{dims: dimensions}
	if _, err := embed.Execute(ctx, getFirst()); err != nil {
		t.Fatal(err)
	}
	for _, chunk := range mine {
		if len(index.getVectors(chunk)) != 1 {
			t.Fatalf("chunk %d of its own vault carries %d vectors", chunk.id, len(index.getVectors(chunk)))
		}
	}
	for _, chunk := range theirs {
		if len(index.getVectors(chunk)) != 0 {
			t.Fatalf("embedding the first vault gave chunk %d of the second a vector", chunk.id)
		}
	}

	if _, err := embed.Execute(ctx, getSecond()); err != nil {
		t.Fatal(err)
	}
	for _, chunk := range append(append([]storedChunk{}, mine...), theirs...) {
		if len(index.getVectors(chunk)) != 1 {
			t.Fatalf("chunk %d carries %d vectors after both vaults were embedded",
				chunk.id, len(index.getVectors(chunk)))
		}
	}
}

func TestEmbeddingWhileAnotherProcessClaimsVaultReportsBusy(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindFile, bookOf(t, "A Book", words(getSanskrit(), 400)), 1)
	small := cutBooks(t, index, shelf, getFirst())

	claim := &fakeClaim{isIndexed: true}

	model := &embedder{dims: dimensions}
	embed := newEmbed(vaults{getFirst().ID: shelf}, index, index)
	embed.Claim = claim
	embed.Embedder = model
	embed.BatchCharacters = 4000

	res, err := embed.Execute(ctx, getFirst())
	if err != nil {
		t.Fatalf("a claimed vault is not an error: %v", err)
	}
	if !res.IsBusy {
		t.Errorf("res.IsBusy = false, want true")
	}
	if res.Embedded != 0 || len(index.vectors) != 0 {
		t.Errorf("embedded %d chunks while vault was claimed", res.Embedded)
	}
	if model.calls != 0 {
		t.Errorf("model was called %d times while vault was claimed", model.calls)
	}

	// Once released, embedding proceeds and completes.
	if err := claim.release(); err != nil {
		t.Fatal(err)
	}
	secondRes, err := embed.Execute(ctx, getFirst())
	if err != nil {
		t.Fatal(err)
	}
	if secondRes.IsBusy {
		t.Errorf("secondRes.IsBusy = true, want false")
	}
	if secondRes.Embedded != len(small) {
		t.Errorf("secondRes.Embedded = %d, want %d", secondRes.Embedded, len(small))
	}
}

func TestPipelineConcurrentBatchesWithProgress(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindFile, bookOf(t, "A Book", words(getSanskrit(), 1200), words(getSanskrit(), 1200)), 1)
	small := cutBooks(t, index, shelf, getFirst())

	var progressReports []EmbedResult
	model := &embedder{dims: dimensions}
	embed := newEmbed(vaults{getFirst().ID: shelf}, index, index)
	embed.Embedder = model
	embed.BatchCharacters = 1000
	embed.OnProgress = func(r EmbedResult) {
		progressReports = append(progressReports, r)
	}

	res, err := embed.Execute(ctx, getFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Embedded != len(small) {
		t.Errorf("res.Embedded = %d, want %d", res.Embedded, len(small))
	}
	if len(progressReports) < 2 {
		t.Errorf("got %d progress reports, want >= 2", len(progressReports))
	}
	for i := 1; i < len(progressReports); i++ {
		if progressReports[i].Embedded < progressReports[i-1].Embedded {
			t.Errorf("progress reports not monotonically increasing: report[%d]=%d < report[%d]=%d",
				i, progressReports[i].Embedded, i-1, progressReports[i-1].Embedded)
		}
	}
}

func TestDuplicateChunksInBatchAreDeduplicated(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	repeatedSection := words(getSanskrit(), 200)
	shelf.hold(bookPath, domain.KindFile, bookOf(t, "A Book", repeatedSection, repeatedSection, repeatedSection), 1)
	small := cutBooks(t, index, shelf, getFirst())

	model := &embedder{dims: dimensions}
	embed := newEmbed(vaults{getFirst().ID: shelf}, index, index)
	embed.Embedder = model
	embed.BatchCharacters = 50000

	res, err := embed.Execute(ctx, getFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Embedded != len(small) {
		t.Errorf("res.Embedded = %d, want %d", res.Embedded, len(small))
	}
	uniqueTexts := map[string]struct{}{}
	for _, chunk := range small {
		uniqueTexts[chunk.text] = struct{}{}
	}
	if len(uniqueTexts) >= len(small) {
		t.Fatalf("expected duplicate chunks in test fixture, got %d unique out of %d total", len(uniqueTexts), len(small))
	}
	if len(model.seenCount) != len(uniqueTexts) {
		t.Errorf("model saw %d texts, want %d unique texts", len(model.seenCount), len(uniqueTexts))
	}
	seenSet := map[string]struct{}{}
	for _, text := range model.seenCount {
		if _, exists := seenSet[text]; exists {
			t.Errorf("model saw duplicate text %q", text)
		}
		seenSet[text] = struct{}{}
	}
	if len(index.vectors) != len(small) {
		t.Errorf("%d chunks carry a vector, want %d", len(index.vectors), len(small))
	}
	for _, chunk := range small {
		if len(index.getVectors(chunk)) != 1 {
			t.Fatalf("chunk %d carries %d vectors, want 1", chunk.id, len(index.getVectors(chunk)))
		}
	}
}

// Chunks of recently modified sources are embedded before older sources.
func TestEmbeddingPrioritizesRecentlyModifiedSources(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold("library/old.epub", domain.KindFile, bookOf(t, "Old Book", words(getSanskrit(), 400)), 100)
	shelf.hold("library/recent.epub", domain.KindFile, bookOf(t, "Recent Book", words(getLatin(), 400)), 200)

	extract := newExtract(vaults{getFirst().ID: shelf}, index, index)
	if _, err := extract.Execute(ctx, getFirst()); err != nil {
		t.Fatal(err)
	}

	model := &embedder{dims: dimensions}
	embed := newEmbed(vaults{getFirst().ID: shelf}, index, index)
	embed.Embedder = model
	embed.BatchCharacters = 4000
	if _, err := embed.Execute(ctx, getFirst()); err != nil {
		t.Fatal(err)
	}

	if len(model.seenCount) == 0 {
		t.Fatal("no texts were embedded")
	}
	sawSanskrit := false
	for _, text := range model.seenCount {
		isLatin := strings.Contains(text, "aqua") || strings.Contains(text, "terra")
		isSanskrit := strings.Contains(text, "udyana") || strings.Contains(text, "vrksa")
		if isSanskrit {
			sawSanskrit = true
		}
		if isLatin && sawSanskrit {
			t.Errorf("recent source (latin) was embedded after older source (sanskrit): %q", text)
		}
	}
}

func TestEmbeddingWithConcurrentWorkers(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindFile, bookOf(t, "Concurrent Test Book", words(getSanskrit(), 1000), words(getLatin(), 1000)), 1)

	small := cutBooks(t, index, shelf, getFirst())
	if len(small) == 0 {
		t.Fatal("no chunks created")
	}

	model := &embedder{dims: dimensions}
	embed := newEmbed(vaults{getFirst().ID: shelf}, index, index)
	embed.Embedder = model
	embed.BatchCharacters = 200
	embed.Concurrency = 3

	res, err := embed.Execute(ctx, getFirst())
	if err != nil {
		t.Fatalf("concurrent execution failed: %v", err)
	}

	if res.Embedded != len(small) {
		t.Fatalf("embedded %d chunks, want %d", res.Embedded, len(small))
	}
	if len(index.vectors) != len(small) {
		t.Fatalf("stored %d vectors, want %d", len(index.vectors), len(small))
	}
}

// holdingEmbedder embeds every text except those of a source it is told to
// hold: a call with one blocks until its context ends.
type holdingEmbedder struct {
	*embedder
	holds   func(text string) bool
	started chan struct{}
}

func (h *holdingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if slices.ContainsFunc(texts, h.holds) {
		select {
		case h.started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return h.embedder.Embed(ctx, texts)
}

// progressLog keeps the progress reports of a run, newest last.
type progressLog struct {
	mu      sync.Mutex
	reports []EmbedResult
}

func (l *progressLog) add(r EmbedResult) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reports = append(l.reports, r)
}

func (l *progressLog) getLast() EmbedResult {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.reports) == 0 {
		return EmbedResult{}
	}
	return l.reports[len(l.reports)-1]
}

// While the second of three sources is embedding, the report names that source:
// the first has finished, and the third is only read ahead.
func TestProgressNamesTheSourceBeingEmbedded(t *testing.T) {
	first := getFirst()
	const (
		pathA = "library/a.epub"
		pathB = "library/b.epub"
		pathC = "library/c.epub"
	)
	third := []string{"harbour", "quay", "mast", "keel", "rope", "sail", "buoy", "tide"}
	index, shelf := newStore(), newLibrary()
	shelf.hold(pathA, domain.KindFile, bookOf(t, "A", words(getLatin(), 300)), 300)
	shelf.hold(pathB, domain.KindFile, bookOf(t, "B", words(getSanskrit(), 300)), 200)
	shelf.hold(pathC, domain.KindFile, bookOf(t, "C", words(third, 300)), 100)
	cutBooks(t, index, shelf, first)

	aChunks := 0
	for _, chunk := range index.small(first.ID) {
		if chunk.path == pathA {
			aChunks++
		}
	}
	isB := func(text string) bool { return strings.Contains(text, getSanskrit()[0]) }
	model := &holdingEmbedder{
		embedder: &embedder{dims: dimensions}, holds: isB, started: make(chan struct{}, 1),
	}
	var log progressLog
	embed := newEmbed(vaults{first.ID: shelf}, index, index)
	embed.Embedder = model
	embed.BatchCharacters = 1
	embed.OnProgress = log.add

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ended := make(chan error, 1)
	go func() {
		_, err := embed.Execute(ctx, first)
		ended <- err
	}()

	select {
	case <-model.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the run never reached the second source")
	}
	for range 2000 {
		if log.getLast().Embedded >= aChunks {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := log.getLast().Embedded; got != aChunks {
		t.Fatalf("%d chunks of the first source are stored, want %d", got, aChunks)
	}

	last := log.getLast()
	if last.CurrentSource != pathB {
		t.Errorf("the report names %q while %q is embedding", last.CurrentSource, pathB)
	}
	if !slices.Equal(last.Active, []string{pathB}) {
		t.Errorf("the report lists %v as in flight, want only %q", last.Active, pathB)
	}
	cancel()
	if err := <-ended; !errors.Is(err, context.Canceled) {
		t.Errorf("the run ended with %v, want context.Canceled", err)
	}
}

// queryAware tells a query from a document and counts the queries it was asked for.
type queryAware struct {
	*embedder
	queries int
}

func (q *queryAware) EmbedQueries(ctx context.Context, texts []string) ([][]float32, error) {
	q.queries += len(texts)
	return q.Embed(ctx, texts)
}

func TestEmbeddingTheChunksOfAVaultMakesDocumentVectors(t *testing.T) {
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindFile, bookOf(t, "A Book", words(getSanskrit(), 400)), 1)
	cutBooks(t, index, shelf, getFirst())

	embed := newEmbed(vaults{getFirst().ID: shelf}, index, index)
	aware := &queryAware{embedder: &embedder{dims: dimensions}}
	embed.Embedder = aware
	if _, err := embed.Execute(t.Context(), getFirst()); err != nil {
		t.Fatal(err)
	}
	if aware.calls == 0 {
		t.Fatal("nothing was embedded")
	}
	if aware.queries != 0 {
		t.Errorf("%d chunks were embedded as queries, want every chunk embedded as a document", aware.queries)
	}
}

// A batch whose send is cancelled leaves the reports: they name the source they named before it.
func TestACancelledSendWithdrawsItsBatchFromTheReport(t *testing.T) {
	const (
		next   = "notes/next.md"
		unsent = "library/a.epub"
	)
	index := newStore()
	var log progressLog
	embed := newEmbed(nil, index, index)
	embed.OnProgress = log.add
	res := &EmbedResult{CurrentSource: next}
	track := newTracker(embed, 1, res, &sync.Mutex{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := embed.sendBatches(ctx, embeddingport.EmbeddingModel{Name: "m", Dimensions: dimensions},
		[]domain.Passage{{Source: unsent}}, []string{"a passage of the book"}, track, make(chan preparedBatch))

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("the send ended with %v, want context.Canceled", err)
	}
	last := log.getLast()
	if last.CurrentSource != next || len(last.Active) != 0 {
		t.Errorf("the last report names %q with %v in flight, want %q with none", last.CurrentSource, last.Active, next)
	}
	if !track.isIdle() {
		t.Error("the unsent batch is still listed as in flight")
	}
}
