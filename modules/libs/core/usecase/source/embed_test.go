package source

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// dimensions is the width of the model the tests embed with. One byte per
// dimension and one bit per dimension are two lengths, so a representation
// written in the place of the other is visible.
const dimensions = 32

// cutBooks extracts one vault's books and answers with the chunks that owe a vector.
func cutBooks(t *testing.T, index *store, shelf *library, v domain.Vault) []storedChunk {
	t.Helper()
	if _, err := (Extract{Readers: vaults{string(v.ID): shelf}, Sources: index, Owing: index}).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	small := index.small(string(v.ID))
	if len(small) == 0 {
		t.Fatal("the book was not cut into anything that carries a vector")
	}
	return small
}

func TestEmbeddingCarriesOnWhereItStopped(t *testing.T) {
	// The book is long enough that what owes a vector is asked for more than
	// once, so the answer pages as well as resumes.
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindBook,
		bookOf(t, "A Long Book", words(sanskrit, 4500), words(sanskrit, 4500)), 1)

	small := cutBooks(t, index, shelf, first)
	if len(small) <= chunksPerQuery {
		t.Fatalf("%d chunks owe a vector, which one question answers with: the run would never resume", len(small))
	}

	stopped := &embedder{dims: dimensions, refuse: 3}
	embed := Embed{
		Readers: vaults{string(first.ID): shelf}, Chunks: index, Vectors: index,
		Embedder: stopped, BatchCharacters: 4000,
	}
	if _, err := embed.Execute(ctx, first); err == nil {
		t.Fatal("a model that is not there did not stop the run")
	}

	part := len(index.vectors)
	if part == 0 || part == len(small) {
		t.Fatalf("the interrupted run embedded %d of %d chunks, want some of them", part, len(small))
	}
	// A chunk is written before its vector, so every vector the interrupted run
	// left names a chunk that is there.
	for chunk := range index.vectors {
		if !index.holds(chunk) {
			t.Fatalf("a vector was written for chunk %d, which the index does not hold", chunk)
		}
	}

	embed.Embedder = &embedder{dims: dimensions}
	res, err := embed.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}

	// The two halves of resumption: nothing was embedded twice, and nothing was
	// skipped.
	if want := len(small) - part; res.Embedded != want {
		t.Errorf("the second run embedded %d chunks, want the %d the first did not", res.Embedded, want)
	}
	for _, chunk := range small {
		switch len(index.vectors[chunk.id]) {
		case 1:
		case 0:
			t.Fatalf("chunk %d was skipped by both runs", chunk.id)
		default:
			t.Fatalf("chunk %d was embedded %d times", chunk.id, len(index.vectors[chunk.id]))
		}
	}
	if len(index.vectors) != len(small) {
		t.Errorf("%d chunks carry a vector, want the %d that owed one", len(index.vectors), len(small))
	}

	// Nothing owes anything, so a third run does no work at all.
	third, err := embed.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if third.Owing != 0 || third.Embedded != 0 {
		t.Errorf("a run after everything was embedded reports %+v", third)
	}
}

func TestBothRepresentationsOfAVectorAreWrittenTogether(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindBook, bookOf(t, "A Book", words(sanskrit, 400), words(sanskrit, 400)), 1)
	small := cutBooks(t, index, shelf, first)

	var progress []EmbedResult
	model := &embedder{dims: dimensions}
	res, err := (Embed{
		Readers: vaults{string(first.ID): shelf}, Chunks: index, Vectors: index,
		Embedder: model, BatchCharacters: 2000,
		OnProgress: func(r EmbedResult) { progress = append(progress, r) },
	}).Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if res.Embedded != len(small) {
		t.Errorf("embedded = %d, want the %d chunks that owed a vector", res.Embedded, len(small))
	}

	if len(index.groups) < 2 {
		t.Fatalf("the vectors arrived in %d writes, want a group at a time", len(index.groups))
	}
	for _, group := range index.groups {
		for _, v := range group {
			if v.Kind != port.QuantisedInt8 {
				t.Errorf("chunk %d was stored as %q", v.ChunkID, v.Kind)
			}
			if len(v.Value) != dimensions {
				t.Errorf("chunk %d holds %d bytes for %d dimensions", v.ChunkID, len(v.Value), dimensions)
			}
			if want := (dimensions + 7) / 8; len(v.Coarse) != want {
				t.Errorf("chunk %d holds %d coarse bytes, want %d", v.ChunkID, len(v.Coarse), want)
			}
			if v.Model != model.Model() {
				t.Errorf("chunk %d was stored under %s", v.ChunkID, v.Model)
			}
		}
	}
	for _, chunk := range small {
		if len(index.vectors[chunk.id]) != 1 {
			t.Fatalf("chunk %d carries %d vectors", chunk.id, len(index.vectors[chunk.id]))
		}
	}

	if len(progress) < len(index.groups) {
		t.Errorf("progress was reported %d times for %d groups", len(progress), len(index.groups))
	}
	if last := progress[len(progress)-1]; last.Reading != bookPath {
		t.Errorf("progress says it is reading %q", last.Reading)
	}
}

func TestWithNoEmbedderTheTextIsCutAndNothingFails(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindBook, bookOf(t, "A Book", words(sanskrit, 400)), 1)
	small := cutBooks(t, index, shelf, first)

	res, err := (Embed{Readers: vaults{string(first.ID): shelf}, Chunks: index, Vectors: index}).Execute(ctx, first)
	if err != nil {
		t.Fatalf("a vault with no embedder is not an error: %v", err)
	}
	if res != (EmbedResult{}) {
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
		want    func(t *testing.T, res EmbedResult, owed int)
	}{
		{
			name:    "the file is gone",
			replace: func(_ *testing.T, shelf *library) { delete(shelf.files, bookPath) },
			want: func(t *testing.T, res EmbedResult, owed int) {
				if res.Vanished != owed || res.Embedded != 0 {
					t.Errorf("the run reports %+v, want all %d chunks left alone", res, owed)
				}
			},
		},
		{
			name: "the file is no longer a book",
			replace: func(_ *testing.T, shelf *library) {
				shelf.hold(bookPath, domain.KindBook, []byte("not an archive"), 2)
			},
			want: func(t *testing.T, res EmbedResult, owed int) {
				if res.Vanished != owed || res.Embedded != 0 {
					t.Errorf("the run reports %+v, want all %d chunks left alone", res, owed)
				}
			},
		},
		{
			name: "the text is shorter than it was",
			replace: func(t *testing.T, shelf *library) {
				shelf.hold(bookPath, domain.KindBook, bookOf(t, "A Book", words(sanskrit, 20)), 2)
			},
			want: func(t *testing.T, res EmbedResult, owed int) {
				if res.Displaced == 0 {
					t.Errorf("the run reports %+v, want the places past the end of the text left alone", res)
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := t.Context()
			index, shelf := newStore(), newLibrary()
			shelf.hold(bookPath, domain.KindBook, bookOf(t, "A Book", words(sanskrit, 400)), 1)
			small := cutBooks(t, index, shelf, first)
			c.replace(t, shelf)

			res, err := (Embed{
				Readers: vaults{string(first.ID): shelf}, Chunks: index, Vectors: index,
				Embedder: &embedder{dims: dimensions},
			}).Execute(ctx, first)
			if err != nil {
				t.Fatalf("a source that moved on is not an error: %v", err)
			}
			if res.Owing != len(small) {
				t.Errorf("owing = %d, want the %d chunks with no vector", res.Owing, len(small))
			}
			if res.Embedded+res.Vanished+res.Displaced != res.Owing {
				t.Errorf("the run reports %+v, and every chunk it was given is embedded or accounted for", res)
			}
			c.want(t, res, len(small))
		})
	}
}

func TestEmbeddingStaysInsideItsVault(t *testing.T) {
	ctx := t.Context()
	index := newStore()
	shelves := vaults{string(first.ID): newLibrary(), string(second.ID): newLibrary()}
	shelves[string(first.ID)].hold("library/sanskrit.epub", domain.KindBook,
		bookOf(t, "Sanskrit", words(sanskrit, 400)), 1)
	shelves[string(second.ID)].hold("library/latin.epub", domain.KindBook,
		bookOf(t, "Latin", words(latin, 400)), 1)

	extract := Extract{Readers: shelves, Sources: index, Owing: index}
	for _, v := range []domain.Vault{first, second} {
		if _, err := extract.Execute(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	mine, theirs := index.small(string(first.ID)), index.small(string(second.ID))
	if len(mine) == 0 || len(theirs) == 0 {
		t.Fatal("both vaults have to hold chunks for this to say anything")
	}

	embed := Embed{Readers: shelves, Chunks: index, Vectors: index, Embedder: &embedder{dims: dimensions}}
	if _, err := embed.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	for _, chunk := range mine {
		if len(index.vectors[chunk.id]) != 1 {
			t.Fatalf("chunk %d of its own vault carries %d vectors", chunk.id, len(index.vectors[chunk.id]))
		}
	}
	for _, chunk := range theirs {
		if len(index.vectors[chunk.id]) != 0 {
			t.Fatalf("embedding the first vault gave chunk %d of the second a vector", chunk.id)
		}
	}

	if _, err := embed.Execute(ctx, second); err != nil {
		t.Fatal(err)
	}
	for _, chunk := range append(append([]storedChunk{}, mine...), theirs...) {
		if len(index.vectors[chunk.id]) != 1 {
			t.Fatalf("chunk %d carries %d vectors after both vaults were embedded",
				chunk.id, len(index.vectors[chunk.id]))
		}
	}
}

func TestASourceStandingOnAReadingIsEmbeddedFromIt(t *testing.T) {
	// A source that stands on what a model read in it holds no text of its own
	// that a chunk is a place in. Reaching that text is the store's, and an
	// embedder without one passes the chunk over and says nothing.
	ctx := t.Context()
	index, shelf, made := newStore(), newLibrary(), newShelf()

	raw := bookOf(t, "Scanned", words(sanskrit, 400))
	shelf.hold(bookPath, domain.KindBook, raw, 1)
	read, _, _ := ocr.Write([]ocr.Page{{Index: 0, Blocks: []ocr.Block{{Label: "text", Text: words(sanskrit, 400)}}}})
	if err := made.Write(ctx, text.Artifact("ocr", text.Fingerprint(raw)), read); err != nil {
		t.Fatal(err)
	}

	extract := Extract{Readers: vaults{string(first.ID): shelf}, Sources: index, Owing: index, Derived: made}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	if index.sources[string(first.ID)][bookPath].TextFrom == "" {
		t.Fatal("the source does not stand on a reading, so embedding it proves nothing")
	}
	small := index.small(string(first.ID))
	if len(small) == 0 {
		t.Fatal("the book was not cut into anything that carries a vector")
	}

	embed := Embed{
		Readers: vaults{string(first.ID): shelf}, Derived: made, Chunks: index, Vectors: index,
		Embedder: &embedder{dims: dimensions}, BatchCharacters: 4000,
	}
	if _, err := embed.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	if len(index.vectors) != len(small) {
		t.Errorf("%d of %d chunks carry a vector", len(index.vectors), len(small))
	}
}
