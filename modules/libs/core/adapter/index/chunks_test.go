package index

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

// The two vaults every scoping test here uses. Their content shares nothing, so
// a row that arrives from the wrong one is recognisable.
var (
	first  = domain.Vault{ID: "01FIRST", Name: "first", Path: "/first"}
	second = domain.Vault{ID: "01SECOND", Name: "second", Path: "/second"}
)

// bits is a coarse vector: one bit per dimension over 1024 dimensions. The seed
// fills every byte, so two different seeds are far apart and the same seed is at
// distance zero.
func bits(seed byte) []byte {
	b := make([]byte, 128)
	for i := range b {
		b[i] = seed
	}
	return b
}

// direction is the vector a seed stands for: dimension i takes the sign of the
// bit the coarse vector keeps it in. Quantising it gives back `bits(seed)`, so
// the two representations of a chunk agree.
func direction(seed byte) []float32 {
	v := make([]float32, 1024)
	for i := range v {
		v[i] = -1
		if seed&(1<<(7-uint(i)%8)) != 0 {
			v[i] = 1
		}
	}
	return v
}

// precise is the full-precision half of a vector, one signed byte per dimension.
func precise(v []float32) []byte {
	q := embedding.Bytes(embedding.Normalise(v))
	out := make([]byte, len(q))
	for i, x := range q {
		out[i] = byte(x)
	}
	return out
}

// migrated is the schema, built once for this binary. A test that only needs an
// index to exist is handed a copy of the file.
var migrated = testsupport.NewTemplate(func(ctx context.Context, path string) error {
	db, err := Open(ctx, path)
	if err != nil {
		return err
	}
	return db.Close()
})

func opened(t *testing.T) *DB {
	t.Helper()
	return openedAt(t, filepath.Join(t.TempDir(), "index.db"))
}

// openedAt is an index at the path given, holding both vaults and nothing else.
func openedAt(t *testing.T, path string) *DB {
	t.Helper()
	migrated.CopyTo(t, path)
	db, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	for _, v := range []domain.Vault{first, second} {
		if err := db.Vaults().Register(t.Context(), v.ID); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

// book puts one source in, cuts it into a large chunk with two small ones
// inside it, and embeds the small ones at the seed given.
//
// Every chunk carries the stem of its own path among its words, so a hit says
// which vault it came from.
func book(t *testing.T, db *DB, vault domain.Vault, path string, seed byte) {
	t.Helper()
	ctx := t.Context()
	chunks := db.Chunks()

	stem := strings.TrimSuffix(strings.TrimPrefix(path, "library/"), ".epub")
	if err := chunks.SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, Hash: "hash-" + path, Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	if err := chunks.ReplaceChunks(ctx, vault.ID, "book", path, []chunk.Chunk{{
		Start: 0, Length: 100, Location: "chapter 1",
		Text: stem + " opening " + stem + " middle",
		Small: []chunk.Chunk{
			{Start: 0, Length: 50, Text: stem + " opening"},
			{Start: 50, Length: 50, Text: stem + " middle"},
		},
	}}); err != nil {
		t.Fatal(err)
	}
	vectorise(t, db, vault, seed)
}

// vectorise gives every chunk of a vault that has no vector one at the seed given.
func vectorise(t *testing.T, db *DB, vault domain.Vault, seed byte) {
	t.Helper()
	ctx := t.Context()

	owing, err := db.ChunkQueries().Unembedded(ctx, vault.ID, "model", 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	vectors := make([]chunk.Vector, 0, len(owing))
	for _, p := range owing {
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: hashOf(t, db, p.Chunk), Recipe: "model",
			Value: precise(direction(seed)), Coarse: bits(seed),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatal(err)
	}
}

// hashOf addresses the text one chunk holds, which is what the vector made
// from it is kept under. The real path hashes what it is about to send; a
// fixture reads what the cut already recorded.
func hashOf(t *testing.T, db *DB, chunk int64) []byte {
	t.Helper()

	var held string
	if err := db.write.QueryRowContext(t.Context(),
		`SELECT hash FROM chunks WHERE id = ?`, chunk).Scan(&held); err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	raw, err := hex.DecodeString(held)
	if err != nil || len(raw) == 0 {
		// A fixture that stored no text still needs a key of its own.
		sum := sha256.Sum256([]byte(strconv.FormatInt(chunk, 10)))
		return sum[:]
	}
	return raw
}

func counted(t *testing.T, db *DB, statement string, args ...any) int {
	t.Helper()
	var n int
	if err := db.read.QueryRowContext(t.Context(), statement, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestTheCoarsePassStaysInsideItsVault(t *testing.T) {
	// A nearest-neighbour search answers with the whole table's best k. Both
	// vaults are seeded near enough to either query to be an answer, and each
	// query is the other vault's own vector, so the wrong vault's rows are the
	// nearer ones and a lost filter puts them first.
	ctx := t.Context()
	db := opened(t)
	book(t, db, first, "library/first.epub", 0xfe)
	book(t, db, second, "library/second.epub", 0xff)

	queries := db.ChunkQueries()

	near := func(vault domain.Vault, seed byte) []domain.Passage {
		matches, err := queries.Nearest(ctx, vault.ID, "model", direction(seed), nil, 10, search.DefaultFloor)
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) == 0 {
			t.Fatalf("%s answered nothing at all", vault.Name)
		}
		return matches
	}

	// Asked with the second vault's own vector, the first vault must still
	// answer with its own passages.
	for _, m := range near(first, 0xff) {
		if !strings.HasPrefix(m.Source, "library/first") {
			t.Errorf("the first vault answered with %s, which belongs to the second", m.Source)
		}
	}
	for _, m := range near(second, 0xfe) {
		if !strings.HasPrefix(m.Source, "library/second") {
			t.Errorf("the second vault answered with %s, which belongs to the first", m.Source)
		}
	}
}

func TestASearchByWordsStaysInsideItsVault(t *testing.T) {
	// One database holds every vault, and a full-text match runs across the
	// whole table: the vault is a filter on the match, and a query that forgets
	// it answers with another vault's passages.
	ctx := t.Context()
	db := opened(t)
	book(t, db, first, "library/first.epub", 0x00)
	book(t, db, second, "library/second.epub", 0xff)

	queries := db.ChunkQueries()

	// "opening" is in both vaults, so what separates them is the filter.
	shared, err := queries.Lexical(ctx, first.ID, "opening", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(shared) == 0 {
		t.Fatal("the first vault answered nothing at all")
	}
	for _, p := range shared {
		if !strings.HasPrefix(p.Source, "library/first") {
			t.Errorf("the first vault answered with %s, which belongs to the second", p.Source)
		}
	}

	// A word only the other vault holds is not in this one.
	leaked, err := queries.Lexical(ctx, first.ID, "second", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaked) != 0 {
		t.Errorf("the first vault answered with %+v, which is the second's word", leaked)
	}
}

func TestAHitComesBackAsTheChunkThatIsRead(t *testing.T) {
	// A small chunk is searched and the large chunk enclosing it is read, so a
	// result arrives with enough text around it to be understood.
	ctx := t.Context()
	db := opened(t)
	book(t, db, first, "library/first.epub", 0x00)

	queries := db.ChunkQueries()
	dense, err := queries.Nearest(ctx, first.ID, "model", direction(0x00), nil, 10, search.DefaultFloor)
	if err != nil {
		t.Fatal(err)
	}
	if len(dense) != 2 {
		t.Fatalf("%d matches, want the two small chunks", len(dense))
	}
	// The words half finds the small chunks and the large one, which is one row
	// per chunk of the book.
	lexical, err := queries.Lexical(ctx, first.ID, "first", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lexical) != 3 {
		t.Fatalf("%d matches, want every chunk of the one book", len(lexical))
	}

	for _, p := range append(dense, lexical...) {
		if p.Source != "library/first.epub" || p.Start != 0 || p.Length != 100 {
			t.Errorf("%+v does not read the whole of the chunk enclosing it", p)
		}
		// The large chunk carries the human location, and a hit inside it is
		// read at that place.
		if p.Location != "chapter 1" {
			t.Errorf("a hit came back at the location %q", p.Location)
		}
		if p.ChunkID == "" {
			t.Error("a hit came back naming no chunk, so nothing can be merged on it")
		}
	}
}

// TestAPassageSaysWhatItWasReadOutOf. A list a person runs their eye down draws
// a book and a recording as what they are, and every ranking has to say which
// it answered with.
func TestAPassageSaysWhatItWasReadOutOf(t *testing.T) {
	ctx := t.Context()
	db := opened(t)
	source(t, db, first, "library/talk.epub", domain.KindBook, "opening words")
	source(t, db, first, "talks/lecture.mp3", domain.KindRecording, "opening words")
	source(t, db, first, "Entropy.md", domain.KindNote, "opening words")
	vectorise(t, db, first, 0x00)

	queries := db.ChunkQueries()
	want := map[string]domain.SourceKind{
		"library/talk.epub": domain.KindBook,
		"talks/lecture.mp3": domain.KindRecording,
		"Entropy.md":        domain.KindNote,
	}

	lexical, err := queries.Lexical(ctx, first.ID, "opening", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	named, err := queries.Named(ctx, first.ID, "opening", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	dense, err := queries.Nearest(ctx, first.ID, "model", direction(0x00), nil, 10, search.DefaultFloor)
	if err != nil {
		t.Fatal(err)
	}
	for _, half := range []struct {
		asked string
		found []domain.Passage
	}{{"words", lexical}, {"names", named}, {"meaning", dense}} {
		if len(half.found) == 0 {
			t.Errorf("the %s half answered with nothing, so it says nothing about a kind", half.asked)
		}
		for _, p := range half.found {
			if p.Kind != want[p.Source] {
				t.Errorf("%s came back as %q, want %q", p.Source, p.Kind, want[p.Source])
			}
		}
	}
}

// source is one source of one kind, holding one chunk under a name of its own.
func source(t *testing.T, db *DB, vault domain.Vault, path string, kind domain.SourceKind, text string) {
	t.Helper()
	ctx := t.Context()
	chunks := db.Chunks()
	if err := chunks.SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: string(kind), Size: 1000, MTime: 1, Hash: "hash-" + path, Recipe: "any",
	}); err != nil {
		t.Fatal(err)
	}
	if err := chunks.ReplaceChunks(ctx, vault.ID, string(kind), path, []chunk.Chunk{{
		Start: 0, Length: 100, Location: "opening", Opens: []string{"opening"}, Text: text,
		Small: []chunk.Chunk{{Start: 0, Length: 50, Text: text}},
	}}); err != nil {
		t.Fatal(err)
	}
}

func TestCuttingASourceTwiceDoesNotDoubleIt(t *testing.T) {
	// Nothing cascades when a source is written again: the row survives, so its
	// chunks are cleared by hand, and the rows in both virtual tables with them.

	db := opened(t)
	book(t, db, first, "library/first.epub", 0x00)

	chunks := counted(t, db, `SELECT COUNT(*) FROM chunks`)
	vectors := counted(t, db, `SELECT COUNT(*) FROM chunks_vec`)
	vec := counted(t, db, `SELECT COUNT(*) FROM chunks_vec`)
	fts := counted(t, db, `SELECT COUNT(*) FROM chunks_fts`)
	if chunks != 3 || vectors != 2 || vec != 2 || fts != 3 {
		t.Fatalf("cutting once gave %d chunks, %d vectors, %d rows in the vector index, %d in the full-text index",
			chunks, vectors, vec, fts)
	}

	book(t, db, first, "library/first.epub", 0x00)

	if got := counted(t, db, `SELECT COUNT(*) FROM chunks`); got != chunks {
		t.Errorf("%d chunks after cutting the same source twice, want %d", got, chunks)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != vectors {
		t.Errorf("%d vectors after cutting the same source twice, want %d", got, vectors)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != vec {
		t.Errorf("%d rows in the vector index after cutting the same source twice, want %d", got, vec)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_fts`); got != fts {
		t.Errorf("%d rows in the full-text index after cutting the same source twice, want %d", got, fts)
	}
}

func TestResavingANoteTakesItsChunksWithIt(t *testing.T) {
	// A note is saved with ON CONFLICT DO UPDATE, so its source row survives and
	// nothing cascades. Its chunks describe text that has changed, and one left
	// behind sends a reader to an offset the file no longer has.
	ctx := t.Context()
	db := opened(t)

	note := domain.Note{
		Fingerprint: domain.Fingerprint{Path: "notes/Entropy.md", Size: 14, ModTime: walked},
		Title:       "Entropy",
		Body:        "the first body",
	}
	if err := db.Notes().Save(ctx, first.ID, []domain.Note{note}); err != nil {
		t.Fatal(err)
	}

	// A note is cut the way a book is: one large chunk over the whole of it, and
	// the small chunks inside it that carry the vectors.
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks WHERE parent_id IS NULL`); got != 1 {
		t.Errorf("%d large chunks for one note", got)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks WHERE parent_id IS NOT NULL`); got == 0 {
		t.Fatal("a note has no small chunks, so nothing about it can be embedded")
	}
	vectorise(t, db, first, 0x00)
	was := counted(t, db, `SELECT COUNT(*) FROM chunks_vec`)
	if was == 0 {
		t.Fatal("the note was not embedded, so this test would pass either way")
	}

	note.Fingerprint.Size = 23
	note.Body = "the second body, longer"
	if err := db.Notes().Save(ctx, first.ID, []domain.Note{note}); err != nil {
		t.Fatal(err)
	}

	// What is left is the note as it stands now, cut once.
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks WHERE parent_id IS NULL`); got != 1 {
		t.Errorf("%d large chunks after a note was rewritten", got)
	}
	if got := counted(t, db,
		`SELECT COUNT(*) FROM chunks c WHERE c.parent_id IS NOT NULL
		   AND c.parent_id NOT IN (SELECT id FROM chunks WHERE parent_id IS NULL)`); got != 0 {
		t.Errorf("%d small chunks sit inside a large one that is gone", got)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != 0 {
		t.Errorf("%d vectors describe a note that has been rewritten", got)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != 0 {
		t.Errorf("%d rows in the vector index describe a note that has been rewritten", got)
	}
	// The words of the note as it stands now are what the full-text index holds.
	found, err := db.ChunkQueries().Lexical(ctx, first.ID, "longer", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 || found[0].Source != note.Fingerprint.Path {
		t.Errorf("the rewritten note is not findable by its new words: %+v", found)
	}
	if stale, err := db.ChunkQueries().Lexical(ctx, first.ID, "first", nil, 10, false); err != nil {
		t.Fatal(err)
	} else if len(stale) != 0 {
		t.Errorf("the words of the note before it was rewritten still answer: %+v", stale)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM notes`); got != 1 {
		t.Errorf("%d notes, want the one that was saved twice", got)
	}
}

// A note is searchable by its meaning, which needs the chunks that carry
// vectors. One large chunk and nothing inside it is a note the dense half can
// never return.
func TestANoteIsCutIntoChunksThatCanCarryAVector(t *testing.T) {
	ctx := t.Context()
	db := opened(t)

	body := strings.Repeat("entropy is the measure of disorder in a closed system. ", 8)
	note := domain.Note{
		Fingerprint: domain.Fingerprint{Path: "notes/Entropy.md", Size: int64(len(body)), ModTime: walked},
		Title:       "Entropy",
		Body:        body,
	}
	if err := db.Notes().Save(ctx, first.ID, []domain.Note{note}); err != nil {
		t.Fatal(err)
	}

	owing, err := db.ChunkQueries().Unembedded(ctx, first.ID, "model", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) < 2 {
		t.Fatalf("%d chunks of a note owe a vector, so its text is one chunk", len(owing))
	}
	// Every offset is into the file, so the text of a passage can be read back.
	for _, p := range owing {
		if p.Start < 0 || p.Start+p.Length > int(note.Fingerprint.Size) {
			t.Errorf("a chunk lies outside the file: %+v", p)
		}
	}
}

func TestRemovingASourceLeavesNothingSearchable(t *testing.T) {
	// The text is not stored, so a passage that outlives its source sends the
	// reader to a file that is gone.
	ctx := t.Context()
	db := opened(t)
	book(t, db, first, "library/first.epub", 0x00)
	book(t, db, second, "library/second.epub", 0xff)

	if err := db.Chunks().RemoveSources(ctx, first.ID, "book", []string{"library/first.epub"}); err != nil {
		t.Fatal(err)
	}

	matches, err := db.ChunkQueries().Nearest(ctx, first.ID, "model", direction(0x00), nil, 10, search.DefaultFloor)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("a removed book still answers a search: %+v", matches)
	}
	words, err := db.ChunkQueries().Lexical(ctx, first.ID, "first", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 0 {
		t.Errorf("a removed book still answers by its words: %+v", words)
	}
	// The second vault is untouched: a removal names a vault as well as a path.
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != 2 {
		t.Errorf("%d rows in the vector index, want the second vault's two", got)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_fts`); got != 3 {
		t.Errorf("%d rows in the full-text index, want the second vault's three", got)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM sources`); got != 1 {
		t.Errorf("%d sources, want the second vault's one", got)
	}
}

func TestRemovingANoteTakesItsIndexedRows(t *testing.T) {
	ctx := t.Context()
	db := opened(t)

	note := domain.Note{
		Fingerprint: domain.Fingerprint{Path: "notes/Entropy.md", Size: 6, ModTime: walked},
		Title:       "Entropy",
		Body:        "a body",
	}
	if err := db.Notes().Save(ctx, first.ID, []domain.Note{note}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().ReplaceChunks(ctx, first.ID, "note", note.Fingerprint.Path, []chunk.Chunk{{
		Start: 0, Length: 6, Text: note.Body,
		Small: []chunk.Chunk{{Start: 0, Length: 6, Text: note.Body}},
	}}); err != nil {
		t.Fatal(err)
	}
	vectorise(t, db, first, 0x00)

	if err := db.Notes().Remove(ctx, first.ID, []string{note.Fingerprint.Path}); err != nil {
		t.Fatal(err)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != 0 {
		t.Errorf("%d rows in the vector index outlived the note they describe", got)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_fts`); got != 0 {
		t.Errorf("%d rows in the full-text index outlived the note they describe", got)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM sources`); got != 0 {
		t.Errorf("%d sources outlived the note", got)
	}
}

func TestANoteScanDoesNotSeeABook(t *testing.T) {
	// Notes and books are rows of one table, and a scan decides what vanished by
	// comparing what the index holds against what is on disk. Every question about
	// sources names a kind.
	ctx := t.Context()
	db := opened(t)
	book(t, db, first, "library/first.epub", 0x00)

	note := domain.Note{
		Fingerprint: domain.Fingerprint{Path: "notes/Entropy.md", Size: 20, ModTime: walked},
		Title:       "Entropy",
		Body:        "a body",
	}
	if err := db.Notes().Save(ctx, first.ID, []domain.Note{note}); err != nil {
		t.Fatal(err)
	}

	known, err := db.NoteQueries().Fingerprints(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 1 {
		t.Errorf("a note scan sees %v, want only the note", known)
	}
	books, err := db.ChunkQueries().Fingerprints(ctx, first.ID, "book")
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 1 || books["library/first.epub"].Size != 1000 {
		t.Errorf("a book scan sees %v, want only the book", books)
	}

	// A name resolves to a note, and a book is not one.
	if paths, err := db.NoteQueries().Named(ctx, first.ID, "first"); err != nil {
		t.Fatal(err)
	} else if len(paths) != 0 {
		t.Errorf("the name of a book resolves to %v", paths)
	}

	// Nothing computes a hash for a note, and the column is null.
	var hashed int
	if err := db.read.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sources WHERE kind = 'note' AND hash IS NOT NULL`).Scan(&hashed); err != nil {
		t.Fatal(err)
	}
	if hashed != 0 {
		t.Errorf("%d notes carry a content hash", hashed)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM sources WHERE hash IS NOT NULL`); got != 1 {
		t.Errorf("%d sources carry a content hash, want the book", got)
	}
}

func TestAChunkCannotClaimAnotherVault(t *testing.T) {
	// The vault a chunk carries is what a search constrains on. If it could
	// The vault a chunk carries is what a search constrains on, and the foreign
	// key ties it to its source's vault.
	ctx := t.Context()
	db := opened(t)
	book(t, db, first, "library/first.epub", 0x00)

	var source, mine, theirs int64
	if err := db.read.QueryRowContext(ctx,
		`SELECT s.id, s.vault_id, (SELECT id FROM vaults WHERE id <> s.vault_id)
		 FROM sources s WHERE s.path = 'library/first.epub'`).Scan(&source, &mine, &theirs); err != nil {
		t.Fatal(err)
	}
	_, err := db.write.ExecContext(ctx,
		`INSERT INTO chunks (source_id, vault_id, start, length, parent_id, location)
		 VALUES (?, ?, 0, 10, NULL, NULL)`, source, theirs)
	if err == nil {
		t.Error("a chunk was written into a vault its source does not belong to")
	}
}

func TestWhatIsStaleIsAskedOnThreeKeys(t *testing.T) {
	// Three keys, because three different things go out of date and each one is
	// repaired by different work: the file, the recipe that read it, and the
	// model the vectors came from.
	ctx := t.Context()
	db := opened(t)
	queries := db.ChunkQueries()

	// The file: a source the index holds and has not cut.
	if err := db.Chunks().SaveSource(ctx, first.ID, chunk.Source{
		Path: "library/uncut.epub", Kind: "book", Size: 10, MTime: 1,
	}); err != nil {
		t.Fatal(err)
	}
	book(t, db, first, "library/cut.epub", 0x00)
	book(t, db, second, "library/other.epub", 0xff)

	uncut, err := queries.Unchunked(ctx, first.ID, "book", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(uncut) != 1 || uncut[0] != "library/uncut.epub" {
		t.Errorf("what has not been cut is %v, want the one uncut book", uncut)
	}

	// The recipe: the same books read by a reader that has since changed.
	byOther, err := queries.ByOtherRecipe(ctx, first.ID, "book", []string{"epub-2"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(byOther) != 2 {
		t.Errorf("%v were extracted by a recipe other than epub-2, want both books", byOther)
	}
	for _, path := range byOther {
		if strings.Contains(path, "other") {
			t.Errorf("the first vault answered with %s, which belongs to the second", path)
		}
	}
	same, err := queries.ByOtherRecipe(ctx, first.ID, "book", []string{"epub"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(same) != 1 || same[0] != "library/uncut.epub" {
		t.Errorf("%v were extracted by a recipe other than epub, want only the one with none", same)
	}

	// The model: chunks whose vectors were made by a different one.
	owing, err := queries.Unembedded(ctx, first.ID, "another-model", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) != 2 {
		t.Errorf("%d chunks owe a vector from another model, want the two small chunks", len(owing))
	}
	for _, p := range owing {
		if !strings.HasPrefix(p.Path, "library/cut") {
			t.Errorf("the first vault answered with %s", p.Path)
		}
	}
	if done, err := queries.Unembedded(ctx, first.ID, "model", 0, 10); err != nil {
		t.Fatal(err)
	} else if len(done) != 0 {
		t.Errorf("%d chunks owe a vector from the model that made theirs", len(done))
	}
}

func TestAnAnswerAboutWhatOwesWorkResumes(t *testing.T) {
	// A vault holds more chunks than one batch, and the question is asked again
	// with the last id of the answer before it.
	ctx := t.Context()
	db := opened(t)
	book(t, db, first, "library/first.epub", 0x00)
	if _, err := db.write.ExecContext(ctx, `DELETE FROM vectors`); err != nil {
		t.Fatal(err)
	}

	queries := db.ChunkQueries()
	firstBatch, err := queries.Unembedded(ctx, first.ID, "model", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstBatch) != 1 {
		t.Fatalf("a batch of one gave %d", len(firstBatch))
	}
	next, err := queries.Unembedded(ctx, first.ID, "model", firstBatch[0].Chunk, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 1 || next[0].Chunk == firstBatch[0].Chunk {
		t.Errorf("asking again from %d gave %+v", firstBatch[0].Chunk, next)
	}
}

func TestTheChildKeyOfAChunkIsIndexed(t *testing.T) {
	// A cascade finds children by the key they point with, and reads the whole
	// table for every parent removed when that key has no index. Which is not
	// something a query plan shows: the delete is a subprogram of the parent's,
	// so this asks the schema instead.
	ctx := t.Context()
	db := opened(t)

	for column, want := range map[string]string{"parent_id": "chunks_by_parent", "source_id": "chunks_by_source"} {
		if !leads(ctx, t, db, "chunks", column) {
			t.Errorf("no index of chunks leads with %s, so %s is missing", column, want)
		}
	}
}

// leads says some index of a table has the column first.
func leads(ctx context.Context, t *testing.T, db *DB, table, column string) bool {
	t.Helper()
	rows, err := db.read.QueryContext(ctx,
		`SELECT i.name FROM pragma_index_list(?) i
		 JOIN pragma_index_info(i.name) c
		 WHERE c.seqno = 0 AND c.name = ?`, table, column)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	found := rows.Next()
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return found
}

// How far embedding has got, and what is left to embed, are two answers about
// one population. They have to agree on which chunks that is, or the total is
// never reached and a bar stops short of the end for ever.
func TestHowFarAndWhatIsLeftAgreeOnWhatIsCounted(t *testing.T) {
	ctx := t.Context()
	db := opened(t)
	vault := first

	if err := db.Chunks().SaveSource(ctx, vault.ID, chunk.Source{
		Path: "library/one.epub", Kind: "book", Size: 1000, MTime: 1,
		Hash: "hash-one", Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	// Two large chunks, each holding two small ones. Only the small ones ever
	// carry a vector.
	large := make([]chunk.Chunk, 0, 2)
	for _, at := range []int{0, 100} {
		large = append(large, chunk.Chunk{
			Start: at, Length: 100, Location: "chapter",
			Text: "whole",
			Small: []chunk.Chunk{
				{Start: at, Length: 50, Text: "first half"},
				{Start: at + 50, Length: 50, Text: "second half"},
			},
		})
	}
	if err := db.Chunks().ReplaceChunks(ctx, vault.ID, "book", "library/one.epub", large); err != nil {
		t.Fatal(err)
	}

	held, embedded, err := db.ChunkQueries().Progress(ctx, vault.ID, "model")
	if err != nil {
		t.Fatal(err)
	}
	owing, err := db.ChunkQueries().Unembedded(ctx, vault.ID, "model", 0, 1000)
	if err != nil {
		t.Fatal(err)
	}

	if embedded != 0 {
		t.Errorf("nothing is embedded yet, and %d are counted", embedded)
	}
	// The invariant: everything held either carries a vector or owes one.
	if int(held) != int(embedded)+len(owing) {
		t.Errorf("%d held is not %d embedded and %d owing", held, embedded, len(owing))
	}

	vectorise(t, db, vault, 1)

	held, embedded, err = db.ChunkQueries().Progress(ctx, vault.ID, "model")
	if err != nil {
		t.Fatal(err)
	}
	owing, err = db.ChunkQueries().Unembedded(ctx, vault.ID, "model", 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) != 0 {
		t.Errorf("%d still owe a vector", len(owing))
	}
	// The total is reached. A count that never reaches it is the defect this
	// asks about.
	if held != embedded {
		t.Errorf("%d held, %d embedded", held, embedded)
	}
	if held == 0 {
		t.Error("nothing was counted at all")
	}
}

// cutInto records one book and makes its chunks the small chunks named, each
// under a large chunk of its own.
func cutInto(t *testing.T, db *DB, vault domain.Vault, path string, texts ...string) {
	t.Helper()
	ctx := t.Context()

	if err := db.Chunks().SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, Hash: "hash-" + path, Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	cut := make([]chunk.Chunk, 0, len(texts))
	for i, text := range texts {
		cut = append(cut, chunk.Chunk{
			Start: i * 100, Length: 100, Text: path + " " + text,
			Small: []chunk.Chunk{{Start: i * 100, Length: 50, Text: text}},
		})
	}
	if err := db.Chunks().ReplaceChunks(ctx, vault.ID, "book", path, cut); err != nil {
		t.Fatal(err)
	}
}

// TestAChunkThatWentIsWrittenNoVectorAndStopsNothing. A save cuts a note again
// while a pass is making vectors out of what it was told owed one a moment ago.
func TestAChunkThatWentIsWrittenNoVectorAndStopsNothing(t *testing.T) {
	ctx := t.Context()
	db := opened(t)

	cutInto(t, db, first, "library/kept.epub", "kept passage")
	cutInto(t, db, first, "library/recut.epub", "the passage as it was")

	// What a pass is given, before anything moves under it.
	owing, err := db.ChunkQueries().Unembedded(ctx, first.ID, "model", 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) != 2 {
		t.Fatalf("%d chunks owe a vector, want the two that were cut", len(owing))
	}

	cutInto(t, db, first, "library/recut.epub", "the passage as it is now")

	vectors := make([]chunk.Vector, 0, len(owing))
	for _, p := range owing {
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: hashOf(t, db, p.Chunk), Recipe: "model",
			Value: bits(1), Coarse: bits(1),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatalf("a chunk that went stopped the pass: %v", err)
	}

	// Both halves of a vector follow the chunk: the one still held carries
	// both, the one that went carries neither.
	var gone int
	for _, p := range owing {
		held := counted(t, db, `SELECT count(*) FROM chunks WHERE id = ?`, p.Chunk)
		if held == 0 {
			gone++
		}
		for _, half := range []string{
			`SELECT count(*) FROM chunks_vec WHERE chunk_id = ?`,
			`SELECT count(*) FROM chunks_vec WHERE chunk_id = ?`,
		} {
			if got := counted(t, db, half, p.Chunk); got != held {
				t.Errorf("chunk %d is held %d times and answers %d to %s", p.Chunk, held, got, half)
			}
		}
	}
	if gone != 1 {
		t.Fatalf("%d of the chunks the pass was given went, want the one that was cut again", gone)
	}
}

func TestAChunkTooFarFromTheQueryIsNoAnswer(t *testing.T) {
	// A nearest-neighbour query answers with k rows whatever was asked. What a
	// vault holds nothing near is not an answer, and the floor is what says so.
	ctx := t.Context()
	db := opened(t)
	book(t, db, first, "library/first.epub", 0x00)
	queries := db.ChunkQueries()

	far, err := queries.Nearest(ctx, first.ID, "model", direction(0xff), nil, 10, search.DefaultFloor)
	if err != nil {
		t.Fatal(err)
	}
	if len(far) != 0 {
		t.Errorf("a vault pointing the other way answered with %v", far)
	}

	// The same vault, asked what it does hold.
	near, err := queries.Nearest(ctx, first.ID, "model", direction(0x00), nil, 10, search.DefaultFloor)
	if err != nil {
		t.Fatal(err)
	}
	if len(near) != 2 {
		t.Errorf("%d answers, want the book's two small chunks", len(near))
	}
}

func TestTheFullPrecisionVectorsDecideTheOrder(t *testing.T) {
	// The two representations of a chunk are written apart here, so the bits say
	// one thing and the full-precision vectors another. The coarse pass keeps
	// several times what is asked for, and what leaves is the one the real
	// vectors put first.
	ctx := t.Context()
	db := opened(t)
	cutInto(t, db, first, "library/coarse.epub", "the passage the bits prefer")
	cutInto(t, db, first, "library/true.epub", "the passage the vectors prefer")

	owing, err := db.ChunkQueries().Unembedded(ctx, first.ID, "model", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) != 2 {
		t.Fatalf("%d chunks owe a vector, want the two small chunks", len(owing))
	}
	if err := db.Chunks().SaveVectors(ctx, []chunk.Vector{{
		Chunk: owing[0].Chunk, Hash: hashOf(t, db, owing[0].Chunk), Recipe: "model",
		Coarse: bits(0xff), Value: precise(direction(0xfe)),
	}, {
		Chunk: owing[1].Chunk, Hash: hashOf(t, db, owing[1].Chunk), Recipe: "model",
		Coarse: bits(0xfe), Value: precise(direction(0xff)),
	}}); err != nil {
		t.Fatal(err)
	}

	near, err := db.ChunkQueries().Nearest(ctx, first.ID, "model", direction(0xff), nil, 1, search.DefaultFloor)
	if err != nil {
		t.Fatal(err)
	}
	if len(near) != 1 {
		t.Fatalf("%d answers, want the one that was asked for", len(near))
	}
	if near[0].Source != "library/true.epub" {
		t.Errorf("the answer is %s, which is the one the bits put first", near[0].Source)
	}
}

// A vector outlives the chunk that asked for it.
//
// Chunks are renumbered by every cut and by every rebuild of the index. What a
// model made is kept by the text it read, so a vault whose rows all went and
// came back again is not bought a second time.
func TestAVectorIsKeptByTheTextItWasMadeFrom(t *testing.T) {
	ctx := t.Context()
	db := opened(t)
	cutInto(t, db, first, "library/kept.epub", "the passage that was paid for")

	owing, err := db.ChunkQueries().Unembedded(ctx, first.ID, "model", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) == 0 {
		t.Fatal("nothing owes a vector")
	}

	text := "the passage that was paid for"
	sum := sha256.Sum256([]byte(text))
	value := precise(direction(0x11))
	if err := db.Chunks().SaveVectors(ctx, []chunk.Vector{{
		Chunk: owing[0].Chunk, Hash: sum[:], Recipe: "a recipe",
		Coarse: bits(0x11), Value: value,
	}}); err != nil {
		t.Fatal(err)
	}

	// Everything a rebuild reaches: the chunks, their vectors, the coarse index.
	for _, statement := range []string{`DELETE FROM sources`, `DELETE FROM chunks_vec`} {
		if _, err := db.write.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != 0 {
		t.Fatalf("%d vectors survived a rebuild of the chunks, want none", got)
	}

	kept, err := db.ChunkQueries().Kept(ctx, "a recipe", [][]byte{sum[:]})
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 1 {
		t.Fatalf("the index kept %d vectors for text it paid for, want 1", len(kept))
	}
	if got := kept[hex.EncodeToString(sum[:])]; !bytes.Equal(got, value) {
		t.Errorf("what was kept is %d bytes, want the %d that were paid for", len(got), len(value))
	}

	// Another recipe is another vector, and this one was never made.
	other, err := db.ChunkQueries().Kept(ctx, "another recipe", [][]byte{sum[:]})
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Errorf("a vector made under one recipe answered for another: %v", other)
	}
}

// One model's vectors are not answers to another model's question.
//
// Two models of the same width write blobs of the same size, and a rerank that
// reads both is comparing directions two models chose for themselves. The
// similarity means nothing and the floor lets it through.
func TestAVectorOfAnotherModelIsNoAnswer(t *testing.T) {
	ctx := t.Context()
	db := opened(t)
	cutInto(t, db, first, "library/first.epub", "a passage two models read")
	queries := db.ChunkQueries()

	owing, err := queries.Unembedded(ctx, first.ID, "another-model", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) == 0 {
		t.Fatal("nothing owes a vector from the other model")
	}
	made := make([]chunk.Vector, 0, len(owing))
	for _, p := range owing {
		made = append(made, chunk.Vector{
			Chunk: p.Chunk, Hash: hashOf(t, db, p.Chunk), Recipe: "another-model",
			Coarse: bits(0x00), Value: precise(direction(0x00)),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, made); err != nil {
		t.Fatal(err)
	}

	// The coarse pass answers, because the bits are there. What is asked of it
	// afterwards is the model's own, and this vector is another model's.
	near, err := queries.Nearest(ctx, first.ID, "model", direction(0x00), nil, 10, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(near) != 0 {
		t.Errorf("a query of one model was answered by another model's vectors: %v", near)
	}

	// Asked of the model that made them, the same rows answer.
	its, err := queries.Nearest(ctx, first.ID, "another-model", direction(0x00), nil, 10, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(its) == 0 {
		t.Error("a model's own vectors did not answer its own question")
	}
}

// Text a source stopped holding takes its vector with it; a source that went
// takes nothing.
//
// A cut says exactly which text this source used to hold and does not any more,
// and that is the one moment the answer is known. A source the vault no longer
// offers is a different thing: a folder that could not be read looks the same
// as one whose files were deleted, and a vector was bought.
func TestTextThatWentTakesItsVectorAndASourceThatWentDoesNot(t *testing.T) {
	ctx := t.Context()
	db := opened(t)
	cutInto(t, db, first, "library/edited.epub", "a passage that will be rewritten")
	cutInto(t, db, first, "library/gone.epub", "a passage in a book that goes")

	owing, err := db.ChunkQueries().Unembedded(ctx, first.ID, "model", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) != 2 {
		t.Fatalf("%d chunks owe a vector, want the two small chunks", len(owing))
	}
	made := make([]chunk.Vector, 0, len(owing))
	for _, p := range owing {
		var hash string
		if err := db.write.QueryRowContext(ctx,
			`SELECT hash FROM chunks WHERE id = ?`, p.Chunk).Scan(&hash); err != nil {
			t.Fatal(err)
		}
		raw, err := hex.DecodeString(hash)
		if err != nil {
			t.Fatal(err)
		}
		made = append(made, chunk.Vector{
			Chunk: p.Chunk, Hash: raw, Recipe: "model",
			Coarse: bits(0x00), Value: precise(direction(0x00)),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, made); err != nil {
		t.Fatal(err)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM vectors`); got != 2 {
		t.Fatalf("%d vectors were kept, want 2", got)
	}

	// The book that went from the vault.
	if err := db.Chunks().RemoveSources(ctx, first.ID, "book", []string{"library/gone.epub"}); err != nil {
		t.Fatal(err)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM vectors`); got != 2 {
		t.Errorf("%d vectors are left after a source went, want both kept", got)
	}

	// The book that was rewritten.
	cutInto(t, db, first, "library/edited.epub", "a passage as it is written now")
	if got := counted(t, db, `SELECT COUNT(*) FROM vectors`); got != 1 {
		t.Errorf("%d vectors are left after the text was replaced, want 1", got)
	}
}

// One vector answers for every chunk holding that text.
//
// Two sources can hold the same passage, and a vector is made once for the
// text. A source that stops holding it says nothing about the others.
func TestAVectorStaysWhileAnyChunkStillHoldsItsText(t *testing.T) {
	ctx := t.Context()
	db := opened(t)
	shared := "the same passage, standing in two books"
	cutInto(t, db, first, "library/one.epub", shared)
	cutInto(t, db, first, "library/two.epub", shared)

	owing, err := db.ChunkQueries().Unembedded(ctx, first.ID, "model", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) != 2 {
		t.Fatalf("%d chunks owe a vector, want two", len(owing))
	}
	made := make([]chunk.Vector, 0, len(owing))
	for _, p := range owing {
		var hash string
		if err := db.write.QueryRowContext(ctx,
			`SELECT hash FROM chunks WHERE id = ?`, p.Chunk).Scan(&hash); err != nil {
			t.Fatal(err)
		}
		raw, err := hex.DecodeString(hash)
		if err != nil {
			t.Fatal(err)
		}
		made = append(made, chunk.Vector{
			Chunk: p.Chunk, Hash: raw, Recipe: "model",
			Coarse: bits(0x00), Value: precise(direction(0x00)),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, made); err != nil {
		t.Fatal(err)
	}
	// One text, one vector, whichever chunk asked for it.
	if got := counted(t, db, `SELECT COUNT(*) FROM vectors`); got != 1 {
		t.Fatalf("%d vectors were kept for one text, want 1", got)
	}

	// One book rewritten; the other still holds the passage.
	cutInto(t, db, first, "library/one.epub", "something else entirely")

	sum := sha256.Sum256([]byte(shared))
	kept, err := db.ChunkQueries().Kept(ctx, "model", [][]byte{sum[:]})
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 1 {
		t.Error("the vector for text a book still holds was taken out")
	}
}

// Vectors carried in under an older name of the same model are taken up.
//
// A migration that carries vectors forward knows only what the rows it reads
// said: which model, and how wide. The recipe in use says more. Where the model
// and the width agree, what was carried was made by the model now in use, and
// it is not bought a second time.

// recognised puts in one source whose text a producer made.
func recognised(t *testing.T, db *DB, vault domain.Vault, path, hash string) {
	t.Helper()

	if err := db.Chunks().SaveSource(t.Context(), vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, Hash: hash, TextFrom: "ocr",
	}); err != nil {
		t.Fatal(err)
	}
}

// What a scan sweeps is its own vault. A source read in another vault is that
// vault's, and this answer holds none of it.
func TestRecognisedStaysInsideItsVault(t *testing.T) {
	db := opened(t)

	recognised(t, db, first, "library/first.pdf", "hash-first")
	recognised(t, db, second, "library/second.pdf", "hash-second")
	book(t, db, first, "library/plain.epub", 1)

	for _, c := range []struct {
		vault domain.Vault
		want  chunk.SourceText
	}{
		{first, chunk.SourceText{Path: "library/first.pdf", Producer: "ocr", Hash: "hash-first"}},
		{second, chunk.SourceText{Path: "library/second.pdf", Producer: "ocr", Hash: "hash-second"}},
	} {
		found, err := db.ChunkQueries().Recognised(t.Context(), c.vault.ID, "book")
		if err != nil {
			t.Fatal(err)
		}
		if len(found) != 1 || found[0] != c.want {
			t.Errorf("%s answers with %v, want only %v", c.vault.Name, found, c.want)
		}
	}
}

// sectioned is a book of three sections, where the words of a section's name
// are said once in its own opening and often in the section after it.
//
// This is the shape a chapter of a scanned book has: the chapter says its
// subject once, in its heading, and a paragraph in the middle of the next
// section says it four times.
func sectioned(t *testing.T, db *DB, vault domain.Vault, path string) {
	t.Helper()
	ctx := t.Context()
	chunks := db.Chunks()

	if err := chunks.SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, Hash: "hash-" + path, Recipe: "pdf",
	}); err != nil {
		t.Fatal(err)
	}
	cut := []chunk.Chunk{
		{
			Start: 0, Length: 100, Location: "Madhavendra Puri",
			Opens: []string{"Madhavendra Puri"},
			Text:  "Madhavendra Puri appeared in the fourteenth century.",
			Small: []chunk.Chunk{{Start: 0, Length: 100, Text: "Madhavendra Puri appeared."}},
		},
		{
			Start: 100, Length: 100, Location: "The Disciplic Succession",
			Opens: []string{"The Disciplic Succession"},
			Text: "Madhavendra Puri was the disciple of Laksmipati. " +
				"Madhavendra Puri's disciples included Isvara Puri. " +
				"Madhavendra Puri is said to be. Madhavendra Puri again.",
			Small: []chunk.Chunk{{Start: 100, Length: 100, Text: "Madhavendra Puri four times over."}},
		},
		{
			Start: 200, Length: 100, Location: "Alice Fenn",
			Opens: []string{"Alice Fenn"},
			Text:  "Alice Fenn met him at the far end of the site.",
			Small: []chunk.Chunk{{Start: 200, Length: 100, Text: "Alice Fenn met him."}},
		},
	}
	if err := chunks.ReplaceChunks(ctx, vault.ID, "book", path, cut); err != nil {
		t.Fatal(err)
	}
}

func TestASectionIsFoundByItsName(t *testing.T) {
	// The words half answers with the paragraph that says the name most often.
	// Asked where a book speaks about a thing, what a person wants is the
	// section about it.
	ctx := t.Context()
	db := opened(t)
	sectioned(t, db, first, "library/chaitanya.pdf")
	queries := db.ChunkQueries()

	lexical, err := queries.Lexical(ctx, first.ID, "Madhavendra Puri", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lexical) == 0 {
		t.Fatal("the words half found nothing")
	}
	if lexical[0].Start == 0 {
		t.Fatal("the words half already answers with the section, and this proves nothing")
	}

	named, err := queries.Named(ctx, first.ID, "Madhavendra Puri", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(named) == 0 {
		t.Fatal("the names half found no section")
	}
	if named[0].Start != 0 {
		t.Errorf("the section came back at %d, and it begins at 0", named[0].Start)
	}
	if named[0].Location != "Madhavendra Puri" {
		t.Errorf("the section came back as %q", named[0].Location)
	}
	if named[0].ChunkID == "" {
		t.Error("the section names no chunk, so nothing can be merged on it")
	}
}

func TestOnlyTheChunkThatOpensASectionCarriesItsName(t *testing.T) {
	// A small chunk standing where a section begins is inside the chunk that
	// opens it. One section named twice is one section answering twice, and the
	// second answer is the same place said again.
	ctx := t.Context()
	db := opened(t)
	sectioned(t, db, first, "library/chaitanya.pdf")

	named, err := db.ChunkQueries().Named(ctx, first.ID, "Madhavendra Puri", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 1 {
		t.Fatalf("%d sections came back, want the one", len(named))
	}
}

func TestASearchByNameStaysInsideItsVault(t *testing.T) {
	ctx := t.Context()
	db := opened(t)
	sectioned(t, db, first, "library/chaitanya.pdf")
	sectioned(t, db, second, "library/chaitanya.pdf")

	named, err := db.ChunkQueries().Named(ctx, first.ID, "Madhavendra Puri", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 1 {
		t.Fatalf("%d sections came back, want the one this vault holds", len(named))
	}
}

func TestASectionNameLeavesWithTheSourceItCameFrom(t *testing.T) {
	// A chunk's number is handed to the next chunk that wants one, so a name
	// left behind by a source that is gone answers for whatever takes it.
	ctx := t.Context()
	db := opened(t)
	chunks := db.Chunks()

	if err := chunks.SaveSource(ctx, first.ID, chunk.Source{
		Path: "library/gone.pdf", Kind: "book", Size: 1000, MTime: 1, Hash: "hash-gone", Recipe: "pdf",
	}); err != nil {
		t.Fatal(err)
	}
	if err := chunks.ReplaceChunks(ctx, first.ID, "book", "library/gone.pdf", []chunk.Chunk{{
		Start: 0, Length: 100, Location: "Thermodynamics",
		Opens: []string{"Thermodynamics"},
		Text:  "Heat moves one way.",
		Small: []chunk.Chunk{{Start: 0, Length: 100, Text: "Heat moves one way."}},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := chunks.RemoveSources(ctx, first.ID, "book", []string{"library/gone.pdf"}); err != nil {
		t.Fatal(err)
	}

	// The next book's chunks take the numbers the first one left, and this book
	// names no section of its own to write over them.
	if err := chunks.SaveSource(ctx, first.ID, chunk.Source{
		Path: "library/next.pdf", Kind: "book", Size: 1000, MTime: 1, Hash: "hash-next", Recipe: "pdf",
	}); err != nil {
		t.Fatal(err)
	}
	if err := chunks.ReplaceChunks(ctx, first.ID, "book", "library/next.pdf", []chunk.Chunk{{
		Start: 0, Length: 100, Location: "Whales",
		Text:  "A whale breathes air.",
		Small: []chunk.Chunk{{Start: 0, Length: 100, Text: "A whale breathes air."}},
	}}); err != nil {
		t.Fatal(err)
	}

	named, err := db.ChunkQueries().Named(ctx, first.ID, "Thermodynamics", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 0 {
		t.Errorf("a name of a source that is gone answers with %+v", named)
	}
}

func TestASectionCutAwayIsNotFoundByItsName(t *testing.T) {
	// A chunk keeps its row through a cut when it says the same thing, so a
	// name is not dropped with the chunk. A section that is no longer there
	// answering by name is a passage opening where nothing begins.
	ctx := t.Context()
	db := opened(t)
	sectioned(t, db, first, "library/chaitanya.pdf")

	// Cut again, and this time nothing opens a section.
	if err := db.Chunks().ReplaceChunks(ctx, first.ID, "book", "library/chaitanya.pdf",
		[]chunk.Chunk{{
			Start: 0, Length: 100, Location: "Madhavendra Puri",
			Text:  "Madhavendra Puri appeared in the fourteenth century.",
			Small: []chunk.Chunk{{Start: 0, Length: 100, Text: "Madhavendra Puri appeared."}},
		}}); err != nil {
		t.Fatal(err)
	}

	named, err := db.ChunkQueries().Named(ctx, first.ID, "Madhavendra Puri", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 0 {
		t.Errorf("%d sections came back from a source that names none", len(named))
	}
}

func TestASectionSurvivesTheWayASourceIsHandedOver(t *testing.T) {
	// A source is written through the port, which is a second shape of a chunk
	// and a place a field is dropped in silence. Everything below here can be
	// right and a section still be unfindable.
	ctx := t.Context()
	db := opened(t)

	if err := db.Sources().SaveExtraction(ctx, first.ID, domain.SourceChunks{
		Source: domain.Source{
			Fingerprint: domain.Fingerprint{
				Path: "library/chaitanya.pdf", Kind: domain.KindBook, Size: 1000, ModTime: walked,
			},
			Hash:   "hash",
			Recipe: "pdf",
		},
		Chunks: []domain.Chunk{{
			Start: 0, Length: 60, Location: "Madhavendra Puri",
			Opens: []string{"Madhavendra Puri"},
			Text:  "Madhavendra Puri appeared in the fourteenth century.",
			Small: []domain.Chunk{{Start: 0, Length: 60, Text: "Madhavendra Puri appeared."}},
		}},
	}); err != nil {
		t.Fatal(err)
	}

	named, err := db.ChunkQueries().Named(ctx, first.ID, "Madhavendra Puri", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 1 {
		t.Fatalf("%d sections came back, want the one the source names", len(named))
	}
	if named[0].Start != 0 {
		t.Errorf("the section came back at %d, and it begins at 0", named[0].Start)
	}
}
