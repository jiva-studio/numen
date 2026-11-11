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

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func getFirst() domain.Vault { return domain.Vault{ID: "01FIRST", Name: "first", Path: "/first"} }

func getSecond() domain.Vault { return domain.Vault{ID: "01SECOND", Name: "second", Path: "/second"} }

func bits(seed byte) []byte {
	b := make([]byte, 128)
	for i := range b {
		b[i] = seed
	}
	return b
}

func getVectorForSeed(seed byte) []float32 {
	v := make([]float32, 1024)
	for i := range v {
		v[i] = -1
		if seed&(1<<(7-uint(i)%8)) != 0 {
			v[i] = 1
		}
	}
	return v
}

func getPreciseBytes(v []float32) []byte {
	q := vector.Bytes(vector.Normalize(v))
	out := make([]byte, len(q))
	for i, x := range q {
		out[i] = byte(x)
	}
	return out
}

func newMigrated() *testsupport.Template {
	return testsupport.NewTemplate(func(ctx context.Context, path string) error {
		db, err := Open(ctx, path)
		if err != nil {
			return err
		}
		return db.Close()
	})
}

func openDB(t *testing.T) *DB {
	t.Helper()
	return openDBAt(t, filepath.Join(t.TempDir(), "index.db"))
}

func openDBAt(t *testing.T, path string) *DB {
	t.Helper()
	newMigrated().CopyTo(t, path)
	db, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	for _, v := range []domain.Vault{getFirst(), getSecond()} {
		if err := db.Vaults().Register(t.Context(), v.ID); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func book(t *testing.T, db *DB, vault domain.Vault, path string, seed byte) {
	t.Helper()
	cutBook(t, db, vault, path)
	saveSeededVectors(t, db, vault, seed)
}

func cutBook(t *testing.T, db *DB, vault domain.Vault, path string) {
	t.Helper()
	ctx := t.Context()
	chunks := db.Chunks()

	stem := strings.TrimSuffix(filepath.Base(path), ".epub")
	if err := chunks.SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, ContentHash: "hash-" + path, Recipe: "epub",
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
}

// testRecipe is the one recipe these tests embed under. A vector is kept under
// the recipe that made it, so a question about one names it.
const testRecipe = "model"

func saveSeededVectors(t *testing.T, db *DB, vault domain.Vault, seed byte) {
	t.Helper()
	ctx := t.Context()

	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(ctx, vault.ID, "model", chunk.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	vectors := make([]chunk.Vector, 0, len(unembedded))
	for _, p := range unembedded {
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: hashOf(t, db, p.Chunk), Recipe: "model",
			Value: getPreciseBytes(getVectorForSeed(seed)), Coarse: bits(seed),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatal(err)
	}
}

func hashOf(t *testing.T, db *DB, chunk int64) []byte {
	t.Helper()

	var target string
	if err := db.write.QueryRowContext(t.Context(),
		`SELECT hash FROM chunks WHERE id = ?`, chunk).Scan(&target); err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	raw, err := hex.DecodeString(target)
	if err != nil || len(raw) == 0 {
		// A fixture that stored no text still needs a key of its own.
		sum := sha256.Sum256([]byte(strconv.FormatInt(chunk, 10)))
		return sum[:]
	}
	return raw
}

func countRows(t *testing.T, db *DB, statement string, args ...any) int {
	t.Helper()
	var n int
	if err := db.read.QueryRowContext(t.Context(), statement, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestTheCoarsePassStaysInsideItsVault(t *testing.T) {
	// A nearest-neighbor search answers with the whole table's best k. Both
	// vaults are seeded near enough to either query to be an answer, and each
	// query is the other vault's own vector, so the wrong vault's rows are the
	// nearer ones and a lost filter puts them first.
	ctx := t.Context()
	db := openDB(t)
	book(t, db, getFirst(), "library/first.epub", 0xfe)
	book(t, db, getSecond(), "library/second.epub", 0xff)

	queries := queriesOf(db)

	nearVal := func(vault domain.Vault, seed byte) []domain.Passage {
		matches, err := queries.FindNearest(ctx, vault.ID, sourcedomain.NearestQuery{Recipe: "model", Vector: getVectorForSeed(seed), Limit: 10, Floor: search.DefaultFloor})
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) == 0 {
			t.Fatalf("%s answered nothing at all", vault.Name)
		}
		return matches
	}

	// Queried with the second vault's vector, the first vault must still
	// return its own passages.
	for _, m := range nearVal(getFirst(), 0xff) {
		if !strings.HasPrefix(m.Source, "library/first") {
			t.Errorf("the first vault answered with %s, which belongs to the second", m.Source)
		}
	}
	for _, m := range nearVal(getSecond(), 0xfe) {
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
	db := openDB(t)
	book(t, db, getFirst(), "library/first.epub", 0x00)
	book(t, db, getSecond(), "library/second.epub", 0xff)

	queries := queriesOf(db)

	// "opening" is in both vaults, so what separates them is the filter.
	shared, err := queries.FindLexical(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "opening", Kinds: nil, Limit: 10})
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
	leaked, err := queries.FindLexical(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "second", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(leaked) != 0 {
		t.Errorf("the first vault answered with %+v, which is the second's word", leaked)
	}
}

func TestAHitReachesBackAsTheChunkThatIsRead(t *testing.T) {
	// A small chunk is searched and the large chunk enclosing it is read, so a
	// result arrives with enough text around it to be understood.
	ctx := t.Context()
	db := openDB(t)
	book(t, db, getFirst(), "library/first.epub", 0x00)

	queries := queriesOf(db)
	dense, err := queries.FindNearest(ctx, getFirst().ID, sourcedomain.NearestQuery{Recipe: "model", Vector: getVectorForSeed(0x00), Limit: 10, Floor: search.DefaultFloor})
	if err != nil {
		t.Fatal(err)
	}
	if len(dense) != 2 {
		t.Fatalf("%d matches, want the two small chunks", len(dense))
	}
	// The words half finds the small chunks, which is one row per small chunk
	// of the book.
	lexical, err := queries.FindLexical(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "first", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(lexical) != 2 {
		t.Fatalf("%d matches, want the two small chunks of the one book", len(lexical))
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

// TestAPassageReturnsWhatItWasReadOutOf checks that every ranking reports the
// source kind (book or recording) of each passage.
func TestAPassageReturnsWhatItWasReadOutOf(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	source(t, db, getFirst(), "library/talk.epub", domain.KindFile, "opening words")
	source(t, db, getFirst(), "talks/lecture.mp3", domain.KindFile, "opening words")
	source(t, db, getFirst(), "Entropy.md", domain.KindNote, "opening words")
	saveSeededVectors(t, db, getFirst(), 0x00)

	queries := queriesOf(db)
	want := map[string]domain.SourceKind{
		"library/talk.epub": domain.KindFile,
		"talks/lecture.mp3": domain.KindFile,
		"Entropy.md":        domain.KindNote,
	}

	lexical, err := queries.FindLexical(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "opening", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	named, err := queries.GetNamedPassages(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "opening", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	dense, err := queries.FindNearest(ctx, getFirst().ID, sourcedomain.NearestQuery{Recipe: "model", Vector: getVectorForSeed(0x00), Limit: 10, Floor: search.DefaultFloor})
	if err != nil {
		t.Fatal(err)
	}
	for _, half := range []struct {
		requested string
		found     []domain.Passage
	}{{"words", lexical}, {"names", named}, {"meaning", dense}} {
		if len(half.found) == 0 {
			t.Errorf("the %s half answered with nothing, so it says nothing about a kind", half.requested)
		}
		for _, p := range half.found {
			if p.Kind != want[p.Source] {
				t.Errorf("%s came back as %q, want %q", p.Source, p.Kind, want[p.Source])
			}
		}
	}
}

func source(t *testing.T, db *DB, vault domain.Vault, path string, kind domain.SourceKind, text string) {
	t.Helper()
	ctx := t.Context()
	chunks := db.Chunks()
	if err := chunks.SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: string(kind), Size: 1000, MTime: 1, ContentHash: "hash-" + path, Recipe: "any",
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

func TestSplittingASourceTwiceDoesNotDoubleIt(t *testing.T) {
	// Writing a source again does not cascade: the row survives, so its
	// chunks and the rows in both virtual tables are cleared explicitly.

	db := openDB(t)
	book(t, db, getFirst(), "library/first.epub", 0x00)

	chunks := countRows(t, db, `SELECT COUNT(*) FROM chunks`)
	vectors := countRows(t, db, `SELECT COUNT(*) FROM vectors`)
	vec := countRows(t, db, `SELECT COUNT(*) FROM chunks_vec`)
	fts := countRows(t, db, `SELECT COUNT(*) FROM chunks_fts`)
	if chunks != 3 || vectors != 2 || vec != 2 || fts != 2 {
		t.Fatalf("cutting once gave %d chunks, %d vectors, %d rows in the vector index, %d in the full-text index",
			chunks, vectors, vec, fts)
	}

	book(t, db, getFirst(), "library/first.epub", 0x00)

	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks`); got != chunks {
		t.Errorf("%d chunks after cutting the same source twice, want %d", got, chunks)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM vectors`); got != vectors {
		t.Errorf("%d vectors after cutting the same source twice, want %d", got, vectors)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != vec {
		t.Errorf("%d rows in the vector index after cutting the same source twice, want %d", got, vec)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks_fts`); got != fts {
		t.Errorf("%d rows in the full-text index after cutting the same source twice, want %d", got, fts)
	}
}

func TestResavingANoteConsumesItsChunksWithIt(t *testing.T) {
	// A note is saved with ON CONFLICT DO UPDATE, so its source row survives and
	// nothing cascades. Its chunks describe changed text, and an outdated chunk
	// points to an offset the file does not have.
	ctx := t.Context()
	db := openDB(t)

	note := domain.Note{
		FileStat: domain.FileStat{Path: "notes/Entropy.md", Size: 14, ModTime: getWalked()},
		Title:    "Entropy",
		Body:     "the first body",
	}
	if err := saveCut(ctx, db, getFirst().ID, []domain.Note{note}); err != nil {
		t.Fatal(err)
	}

	// A note is cut the way a book is: one large chunk over the whole of it, and
	// the small chunks inside it that carry the vectors.
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks WHERE parent_id IS NULL`); got != 1 {
		t.Errorf("%d large chunks for one note", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks WHERE parent_id IS NOT NULL`); got == 0 {
		t.Fatal("a note has no small chunks, so nothing about it can be embedded")
	}
	saveSeededVectors(t, db, getFirst(), 0x00)
	was := countRows(t, db, `SELECT COUNT(*) FROM chunks_vec`)
	if was == 0 {
		t.Fatal("the note was not embedded, so this test would pass either way")
	}

	note.FileStat.Size = 23
	note.Body = "the second body, longer"
	if err := saveCut(ctx, db, getFirst().ID, []domain.Note{note}); err != nil {
		t.Fatal(err)
	}

	// What is left is the note currently now, cut once.
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks WHERE parent_id IS NULL`); got != 1 {
		t.Errorf("%d large chunks after a note was rewritten", got)
	}
	if got := countRows(t, db,
		`SELECT COUNT(*) FROM chunks c WHERE c.parent_id IS NOT NULL
		   AND c.parent_id NOT IN (SELECT id FROM chunks WHERE parent_id IS NULL)`); got != 0 {
		t.Errorf("%d small chunks sit inside a large one that is gone", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != 0 {
		t.Errorf("%d vectors describe a note that has been rewritten", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != 0 {
		t.Errorf("%d rows in the vector index describe a note that has been rewritten", got)
	}
	// The words of the note currently now are what the full-text index holds.
	found, err := db.ChunkSearch().FindLexical(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "longer", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 || found[0].Source != note.FileStat.Path {
		t.Errorf("the rewritten note is not findable by its new words: %+v", found)
	}
	if stale, err := db.ChunkSearch().FindLexical(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "first", Kinds: nil, Limit: 10}); err != nil {
		t.Fatal(err)
	} else if len(stale) != 0 {
		t.Errorf("the words of the note before it was rewritten still answer: %+v", stale)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM notes`); got != 1 {
		t.Errorf("%d notes, want the one that was saved twice", got)
	}
}

// A note is searchable by its meaning, which needs the chunks that carry
// vectors. One large chunk and nothing inside it is a note the dense half can
// never return.
func TestANoteIsSplitIntoChunksThatCanCarryAVector(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)

	body := strings.Repeat("entropy is the measure of disorder in a closed system. ", 8)
	note := domain.Note{
		FileStat: domain.FileStat{Path: "notes/Entropy.md", Size: int64(len(body)), ModTime: getWalked()},
		Title:    "Entropy",
		Body:     body,
	}
	if err := saveCut(ctx, db, getFirst().ID, []domain.Note{note}); err != nil {
		t.Fatal(err)
	}

	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(ctx, getFirst().ID, "model", chunk.Cursor{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) < 2 {
		t.Fatalf("%d chunks of a note need a vector, so its text is one chunk", len(unembedded))
	}
	// Every offset is into the file, so the text of a passage can be read back.
	for _, p := range unembedded {
		if p.Start < 0 || p.Start+p.Length > int(note.FileStat.Size) {
			t.Errorf("a chunk lies outside the file: %+v", p)
		}
	}
}

func TestRemovingASourcePreservesNothingSearchable(t *testing.T) {
	// The text is not stored, so a passage that outlives its source sends the
	// reader to a file that is gone.
	ctx := t.Context()
	db := openDB(t)
	book(t, db, getFirst(), "library/first.epub", 0x00)
	book(t, db, getSecond(), "library/second.epub", 0xff)

	if err := db.Chunks().RemoveSources(ctx, getFirst().ID, "book", []string{"library/first.epub"}); err != nil {
		t.Fatal(err)
	}

	matches, err := queriesOf(db).FindNearest(ctx, getFirst().ID, sourcedomain.NearestQuery{Recipe: "model", Vector: getVectorForSeed(0x00), Limit: 10, Floor: search.DefaultFloor})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("a removed book still answers a search: %+v", matches)
	}
	words, err := db.ChunkSearch().FindLexical(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "first", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 0 {
		t.Errorf("a removed book still answers by its words: %+v", words)
	}
	// The second vault is untouched: a removal names a vault as well as a path.
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != 2 {
		t.Errorf("%d rows in the vector index, want the second vault's two", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks_fts`); got != 2 {
		t.Errorf("%d rows in the full-text index, want the second vault's two", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM sources`); got != 1 {
		t.Errorf("%d sources, want the second vault's one", got)
	}
}

func TestRemovingANoteConsumesItsIndexedRows(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)

	note := domain.Note{
		FileStat: domain.FileStat{Path: "notes/Entropy.md", Size: 6, ModTime: getWalked()},
		Title:    "Entropy",
		Body:     "a body",
	}
	if err := saveCut(ctx, db, getFirst().ID, []domain.Note{note}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().ReplaceChunks(ctx, getFirst().ID, "note", note.FileStat.Path, []chunk.Chunk{{
		Start: 0, Length: 6, Text: note.Body,
		Small: []chunk.Chunk{{Start: 0, Length: 6, Text: note.Body}},
	}}); err != nil {
		t.Fatal(err)
	}
	saveSeededVectors(t, db, getFirst(), 0x00)

	if err := db.Notes().Remove(ctx, getFirst().ID, []string{note.FileStat.Path}); err != nil {
		t.Fatal(err)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != 0 {
		t.Errorf("%d rows in the vector index outlived the note they describe", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks_fts`); got != 0 {
		t.Errorf("%d rows in the full-text index outlived the note they describe", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM sources`); got != 0 {
		t.Errorf("%d sources outlived the note", got)
	}
}

func TestANoteScanDoesNotSeeABook(t *testing.T) {
	// Notes and books are rows of one table, and a scan decides what vanished by
	// comparing what the index holds against what is on disk. Every question about
	// sources names a kind.
	ctx := t.Context()
	db := openDB(t)
	book(t, db, getFirst(), "library/first.epub", 0x00)

	note := domain.Note{
		FileStat: domain.FileStat{Path: "notes/Entropy.md", Size: 20, ModTime: getWalked()},
		Title:    "Entropy",
		Body:     "a body",
	}
	if err := saveCut(ctx, db, getFirst().ID, []domain.Note{note}); err != nil {
		t.Fatal(err)
	}

	recognized, err := db.NoteQueries().GetFileStats(ctx, getFirst().ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 1 {
		t.Errorf("a note scan sees %v, want only the note", recognized)
	}
	books, err := db.ChunkFiles().GetFileStats(ctx, getFirst().ID, "book")
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 1 || books["library/first.epub"].Size != 1000 {
		t.Errorf("a book scan sees %v, want only the book", books)
	}

	// A name resolves to a note, and a book is not one.
	if paths, err := db.NoteQueries().GetNamedPaths(ctx, getFirst().ID, "first"); err != nil {
		t.Fatal(err)
	} else if len(paths) != 0 {
		t.Errorf("the name of a book resolves to %v", paths)
	}

	// Nothing computes a hash for a note, and the column is null.
	var hashed int
	if err := db.read.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sources WHERE kind = 'note' AND content_hash IS NOT NULL`).Scan(&hashed); err != nil {
		t.Fatal(err)
	}
	if hashed != 0 {
		t.Errorf("%d notes carry a content hash", hashed)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM sources WHERE content_hash IS NOT NULL`); got != 1 {
		t.Errorf("%d sources carry a content hash, want the book", got)
	}
}

func TestAChunkCannotClaimAnotherVault(t *testing.T) {
	// The vault a chunk carries is what a search constrains on. If it could
	// The vault a chunk carries is what a search constrains on, and the foreign
	// key ties it to its source's vault.
	ctx := t.Context()
	db := openDB(t)
	book(t, db, getFirst(), "library/first.epub", 0x00)

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

func TestStaleQueries_FileRecipeAndModelKeys(t *testing.T) {
	// Three keys, because three different things go out of date and each one is
	// repaired by different work: the file, the recipe that read it, and the
	// model the vectors came from.
	ctx := t.Context()
	db := openDB(t)
	queries := queriesOf(db)

	// The file: a source the index holds and has not cut.
	if err := db.Chunks().SaveSource(ctx, getFirst().ID, chunk.Source{
		Path: "library/uncut.epub", Kind: "book", Size: 10, MTime: 1,
	}); err != nil {
		t.Fatal(err)
	}
	book(t, db, getFirst(), "library/cut.epub", 0x00)
	book(t, db, getSecond(), "library/other.epub", 0xff)

	uncut, err := queries.GetUnchunkedSources(ctx, getFirst().ID, "book", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(uncut) != 1 || uncut[0] != "library/uncut.epub" {
		t.Errorf("what has not been cut is %v, want the one uncut book", uncut)
	}

	// The recipe: the same books read by a reader that has since changed.
	byOther, err := queries.GetSourcesWithOtherRecipe(ctx, getFirst().ID, "book", []string{"epub-2"}, 10)
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
	same, err := queries.GetSourcesWithOtherRecipe(ctx, getFirst().ID, "book", []string{"epub"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(same) != 1 || same[0] != "library/uncut.epub" {
		t.Errorf("%v were extracted by a recipe other than epub, want only the one with none", same)
	}

	// The model: chunks whose vectors were made by a different one.
	unembedded, err := queries.GetUnembeddedChunks(ctx, getFirst().ID, "another-model", chunk.Cursor{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) != 2 {
		t.Errorf("%d chunks need a vector from another model, want the two small chunks", len(unembedded))
	}
	for _, p := range unembedded {
		if !strings.HasPrefix(p.Path, "library/cut") {
			t.Errorf("the first vault answered with %s", p.Path)
		}
	}
	if isDone, err := queries.GetUnembeddedChunks(ctx, getFirst().ID, "model", chunk.Cursor{}, 10); err != nil {
		t.Fatal(err)
	} else if len(isDone) != 0 {
		t.Errorf("%d chunks need a vector from the model that made theirs", len(isDone))
	}
}

func TestGetUnembeddedChunks_ResumesFromLastID(t *testing.T) {
	// A vault holds more chunks than one batch; the query is repeated with the
	// last id of the previous batch.
	ctx := t.Context()
	db := openDB(t)
	book(t, db, getFirst(), "library/first.epub", 0x00)
	if _, err := db.write.ExecContext(ctx, `DELETE FROM vectors`); err != nil {
		t.Fatal(err)
	}

	queries := queriesOf(db)
	firstBatch, err := queries.GetUnembeddedChunks(ctx, getFirst().ID, "model", chunk.Cursor{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstBatch) != 1 {
		t.Fatalf("a batch of one gave %d", len(firstBatch))
	}
	next, err := queries.GetUnembeddedChunks(ctx, getFirst().ID, "model", chunk.Cursor{MTime: firstBatch[0].MTime, Source: firstBatch[0].Source, Chunk: firstBatch[0].Chunk}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 1 || next[0].Chunk == firstBatch[0].Chunk {
		t.Errorf("asking again from %d gave %+v", firstBatch[0].Chunk, next)
	}
}

func TestTheChildKeyOfAChunkIsIndexed(t *testing.T) {
	// A cascade finds children by their foreign key and scans the whole table for
	// every removed parent when that key has no index. A query plan does not show
	// this: the delete runs as a subprogram of the parent's, so the test checks
	// the schema.
	ctx := t.Context()
	db := openDB(t)

	for column, want := range map[string]string{"parent_id": "chunks_by_parent", "source_id": "chunks_by_source"} {
		if !hasLeadingColumn(ctx, t, db, "chunks", column) {
			t.Errorf("no index of chunks leads with %s, so %s is missing", column, want)
		}
	}
}

func hasLeadingColumn(ctx context.Context, t *testing.T, db *DB, table, column string) bool {
	t.Helper()
	rows, err := db.read.QueryContext(ctx,
		`SELECT i.name FROM pragma_index_list(?) i
		 JOIN pragma_index_info(i.name) c
		 WHERE c.seqno = 0 AND c.name = ?`, table, column)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Error(err)
		}
	}()
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
	db := openDB(t)
	vault := getFirst()

	if err := db.Chunks().SaveSource(ctx, vault.ID, chunk.Source{
		Path: "library/one.epub", Kind: "book", Size: 1000, MTime: 1,
		ContentHash: "hash-one", Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	// Two large chunks, each holding two small ones. Only the small ones ever
	// carry a vector.
	large := make([]chunk.Chunk, 0, 2)
	for _, item := range []int{0, 100} {
		large = append(large, chunk.Chunk{
			Start: item, Length: 100, Location: "chapter",
			Text: "whole",
			Small: []chunk.Chunk{
				{Start: item, Length: 50, Text: "first half"},
				{Start: item + 50, Length: 50, Text: "second half"},
			},
		})
	}
	if err := db.Chunks().ReplaceChunks(ctx, vault.ID, "book", "library/one.epub", large); err != nil {
		t.Fatal(err)
	}

	target, embedded, err := db.ChunkVectors().GetProgress(ctx, vault.ID, "model")
	if err != nil {
		t.Fatal(err)
	}
	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(ctx, vault.ID, "model", chunk.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}

	if embedded != 0 {
		t.Errorf("nothing is embedded yet, and %d are counted", embedded)
	}
	// Invariant: every chunk either has a vector or is awaiting one.
	if int(target) != int(embedded)+len(unembedded) {
		t.Errorf("%d held is not %d embedded and %d unembedded", target, embedded, len(unembedded))
	}

	saveSeededVectors(t, db, vault, 1)

	target, embedded, err = db.ChunkVectors().GetProgress(ctx, vault.ID, "model")
	if err != nil {
		t.Fatal(err)
	}
	unembedded, err = db.ChunkVectors().GetUnembeddedChunks(ctx, vault.ID, "model", chunk.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) != 0 {
		t.Errorf("%d still need a vector", len(unembedded))
	}
	// The embedded count equals the total; a count that stays below it is the
	// defect under test.
	if target != embedded {
		t.Errorf("%d held, %d embedded", target, embedded)
	}
	if target == 0 {
		t.Error("nothing was counted at all")
	}
}

func cutInto(t *testing.T, db *DB, vault domain.Vault, path string, texts ...string) {
	t.Helper()
	ctx := t.Context()

	if err := db.Chunks().SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, ContentHash: "hash-" + path, Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	cutChunks := make([]chunk.Chunk, 0, len(texts))
	for i, text := range texts {
		cutChunks = append(cutChunks, chunk.Chunk{
			Start: i * 100, Length: 100, Text: path + " " + text,
			Small: []chunk.Chunk{{Start: i * 100, Length: 50, Text: text}},
		})
	}
	if err := db.Chunks().ReplaceChunks(ctx, vault.ID, "book", path, cutChunks); err != nil {
		t.Fatal(err)
	}
}

// TestAChunkThatWentIsSavedNoVectorAndStopsNothing checks that a save re-splitting
// a note while a pass is embedding chunks that were pending does not stop the pass.
func TestAChunkThatWentIsSavedNoVectorAndStopsNothing(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)

	cutInto(t, db, getFirst(), "library/kept.epub", "kept passage")
	cutInto(t, db, getFirst(), "library/recut.epub", "the passage as it was")

	// What a pass is given, before anything moves under it.
	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(ctx, getFirst().ID, "model", chunk.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) != 2 {
		t.Fatalf("%d chunks need a vector, want the two that were cut", len(unembedded))
	}

	cutInto(t, db, getFirst(), "library/recut.epub", "the passage as it is now")

	vectors := make([]chunk.Vector, 0, len(unembedded))
	for _, p := range unembedded {
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
	var removed int
	for _, p := range unembedded {
		target := countRows(t, db, `SELECT count(*) FROM chunks WHERE id = ?`, p.Chunk)
		if target == 0 {
			removed++
		}
		if got := countRows(t, db,
			`SELECT count(*) FROM chunks_vec WHERE chunk_id = ?`, p.Chunk); got != target {
			t.Errorf("chunk %d has count %d in %d coarse index rows",
				p.Chunk, target, got)
		}
		if got := countRows(t, db, `SELECT count(*) FROM chunks c
		                            JOIN vectors v ON v.hash = unhex(c.hash) AND v.recipe = ?
		                            WHERE c.id = ?`, testRecipe, p.Chunk); got != target {
			t.Errorf("chunk %d has count %d in %d vector rows",
				p.Chunk, target, got)
		}
	}
	if removed != 1 {
		t.Fatalf("%d of the chunks the pass was given went, want the one that was cut again", removed)
	}
}

func TestAChunkTooFarFromTheQueryIsNoAnswer(t *testing.T) {
	// A nearest-neighbor query returns k rows regardless of distance. A chunk
	// farther than the floor is not a match.
	ctx := t.Context()
	db := openDB(t)
	book(t, db, getFirst(), "library/first.epub", 0x00)
	queries := queriesOf(db)

	far, err := queries.FindNearest(ctx, getFirst().ID, sourcedomain.NearestQuery{Recipe: "model", Vector: getVectorForSeed(0xff), Limit: 10, Floor: search.DefaultFloor})
	if err != nil {
		t.Fatal(err)
	}
	if len(far) != 0 {
		t.Errorf("a vault pointing the other way answered with %v", far)
	}

	// The same vault, queried with a vector it contains.
	val, err := queries.FindNearest(ctx, getFirst().ID, sourcedomain.NearestQuery{Recipe: "model", Vector: getVectorForSeed(0x00), Limit: 10, Floor: search.DefaultFloor})
	if err != nil {
		t.Fatal(err)
	}
	if len(val) != 2 {
		t.Errorf("%d answers, want the book's two small chunks", len(val))
	}
}

func TestTheFullPrecisionVectorsDecideTheOrder(t *testing.T) {
	// The two representations of a chunk are written separately here, so the bits
	// and the full-precision vectors disagree. The coarse pass keeps several
	// times the requested count, and the result is the chunk that the full
	// vectors rank first.
	ctx := t.Context()
	db := openDB(t)
	cutInto(t, db, getFirst(), "library/coarse.epub", "the passage the bits prefer")
	cutInto(t, db, getFirst(), "library/true.epub", "the passage the vectors prefer")

	unembedded := getTwoUnembedded(t, db)
	for _, p := range unembedded {
		var vec chunk.Vector
		if p.Path == "library/coarse.epub" {
			vec = chunk.Vector{
				Chunk: p.Chunk, Hash: hashOf(t, db, p.Chunk), Recipe: "model",
				Coarse: bits(0xff), Value: getPreciseBytes(getVectorForSeed(0xfe)),
			}
		} else {
			vec = chunk.Vector{
				Chunk: p.Chunk, Hash: hashOf(t, db, p.Chunk), Recipe: "model",
				Coarse: bits(0xfe), Value: getPreciseBytes(getVectorForSeed(0xff)),
			}
		}
		if err := db.Chunks().SaveVectors(ctx, []chunk.Vector{vec}); err != nil {
			t.Fatal(err)
		}
	}

	data, err := queriesOf(db).FindNearest(ctx, getFirst().ID, sourcedomain.NearestQuery{Recipe: "model", Vector: getVectorForSeed(0xff), Limit: 1, Floor: search.DefaultFloor})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 {
		t.Fatalf("%d answers, want the one that was asked for", len(data))
	}
	if data[0].Source != "library/true.epub" {
		t.Errorf("the answer is %s, which is the one the bits put first", data[0].Source)
	}
}

// A vector outlives the chunk it was created for.
//
// Chunk ids are renumbered by every split and every index rebuild. Vectors are
// keyed by the text they were made from, so a vault whose rows are all deleted
// and recreated does not recompute them.
func TestAVectorIsCachedByTheTextItWasMadeFrom(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	cutInto(t, db, getFirst(), "library/kept.epub", "the passage that was paid for")

	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(ctx, getFirst().ID, "model", chunk.Cursor{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) == 0 {
		t.Fatal("nothing needs a vector")
	}

	text := "the passage that was paid for"
	sum := sha256.Sum256([]byte(text))
	value := getPreciseBytes(getVectorForSeed(0x11))
	if err := db.Chunks().SaveVectors(ctx, []chunk.Vector{{
		Chunk: unembedded[0].Chunk, Hash: sum[:], Recipe: "a recipe",
		Coarse: bits(0x11), Value: value,
	}}); err != nil {
		t.Fatal(err)
	}

	// Everything a rebuild affects: the chunks, their vectors, the coarse index.
	for _, statement := range []string{`DELETE FROM sources`, `DELETE FROM chunks_vec`} {
		if _, err := db.write.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chunks_vec`); got != 0 {
		t.Fatalf("%d vectors survived a rebuild of the chunks, want none", got)
	}

	retained, err := db.ChunkVectors().GetKeptVectors(ctx, "a recipe", [][]byte{sum[:]})
	if err != nil {
		t.Fatal(err)
	}
	if len(retained) != 1 {
		t.Fatalf("the index kept %d vectors for text it paid for, want 1", len(retained))
	}
	if got := retained[hex.EncodeToString(sum[:])]; !bytes.Equal(got, value) {
		t.Errorf("what was kept is %d bytes, want the %d that were paid for", len(got), len(value))
	}

	// Another recipe is another vector, and this one was never made.
	other, err := db.ChunkVectors().GetKeptVectors(ctx, "another recipe", [][]byte{sum[:]})
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
	db := openDB(t)
	cutInto(t, db, getFirst(), "library/first.epub", "a passage two models read")
	queries := queriesOf(db)

	unembedded, err := queries.GetUnembeddedChunks(ctx, getFirst().ID, "another-model", chunk.Cursor{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) == 0 {
		t.Fatal("nothing needs a vector from the other model")
	}
	made := make([]chunk.Vector, 0, len(unembedded))
	for _, p := range unembedded {
		made = append(made, chunk.Vector{
			Chunk: p.Chunk, Hash: hashOf(t, db, p.Chunk), Recipe: "another-model",
			Coarse: bits(0x00), Value: getPreciseBytes(getVectorForSeed(0x00)),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, made); err != nil {
		t.Fatal(err)
	}

	// The coarse pass matches because the bits are present. The follow-up
	// full-precision query is model-specific, and this vector belongs to another model.
	itemVal, err := queries.FindNearest(ctx, getFirst().ID, sourcedomain.NearestQuery{Recipe: "model", Vector: getVectorForSeed(0x00), Limit: 10, Floor: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if len(itemVal) != 0 {
		t.Errorf("a query of one model was answered by another model's vectors: %v", itemVal)
	}

	// Queried with the model that created them, the same rows match.
	its, err := queries.FindNearest(ctx, getFirst().ID, sourcedomain.NearestQuery{Recipe: "another-model", Vector: getVectorForSeed(0x00), Limit: 10, Floor: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if len(its) == 0 {
		t.Error("a model's own vectors did not answer its own question")
	}
}

// Text a source stopped containing removes its vector; a removed source removes nothing.
// A split knows which text the source held before and not now. A source the
// vault does not list is ambiguous: an unreadable folder looks like deleted
// files, and recomputing the vector is costly.
// files were deleted, and recomputing the vector is costly.
func TestTextThatWentConsumesItsVectorAndASourceThatWentDoesNot(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	cutInto(t, db, getFirst(), "library/edited.epub", "a passage that will be rewritten")
	cutInto(t, db, getFirst(), "library/gone.epub", "a passage in a book that goes")

	saveVectorPerChunk(t, db)
	if got := countRows(t, db, `SELECT COUNT(*) FROM vectors`); got != 2 {
		t.Fatalf("%d vectors were kept, want 2", got)
	}

	// The book that went from the vault.
	if err := db.Chunks().RemoveSources(ctx, getFirst().ID, "book", []string{"library/gone.epub"}); err != nil {
		t.Fatal(err)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM vectors`); got != 2 {
		t.Errorf("%d vectors are left after a source went, want both kept", got)
	}

	// The book that was rewritten.
	cutInto(t, db, getFirst(), "library/edited.epub", "a passage as it is written now")
	if got := countRows(t, db, `SELECT COUNT(*) FROM vectors`); got != 1 {
		t.Errorf("%d vectors are left after the text was replaced, want 1", got)
	}
}

// One vector serves every chunk containing that text.
//
// Two sources can contain the same passage, and a vector is created once per
// text. A source that stops containing it does not affect the others.
func TestAVectorStaysWhileAnyChunkStillContainsItsText(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	shared := "the same passage, standing in two books"
	cutInto(t, db, getFirst(), "library/one.epub", shared)
	cutInto(t, db, getFirst(), "library/two.epub", shared)

	saveVectorPerChunk(t, db)
	// One text, one vector, regardless of which chunk requested it.
	if got := countRows(t, db, `SELECT COUNT(*) FROM vectors`); got != 1 {
		t.Fatalf("%d vectors were kept for one text, want 1", got)
	}

	// One book rewritten; the other still holds the passage.
	cutInto(t, db, getFirst(), "library/one.epub", "something else entirely")

	sum := sha256.Sum256([]byte(shared))
	retained, err := db.ChunkVectors().GetKeptVectors(ctx, "model", [][]byte{sum[:]})
	if err != nil {
		t.Fatal(err)
	}
	if len(retained) != 1 {
		t.Error("the vector for text a book still holds was taken out")
	}
}

// Vectors carried in under an older name of the same model are taken up.
//
// A migration that carries vectors forward has only the model and the
// dimension recorded in the rows it reads. The recipe in use adds more
// information: when the model and dimension match, the migrated vectors were
// created by the model currently in use and are not recomputed.

func saveRecognizedSource(t *testing.T, db *DB, vault domain.Vault, path, hash string) {
	t.Helper()

	if err := db.Chunks().SaveSource(t.Context(), vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, ContentHash: hash, Producer: "ocr",
	}); err != nil {
		t.Fatal(err)
	}
}

// What a scan sweeps is its own vault. A source read in another vault is that
// vault's, and this answer holds none of it.
func TestRecognizedStaysInsideItsVault(t *testing.T) {
	db := openDB(t)

	saveRecognizedSource(t, db, getFirst(), "library/first.pdf", "hash-first")
	saveRecognizedSource(t, db, getSecond(), "library/second.pdf", "hash-second")
	book(t, db, getFirst(), "library/plain.epub", 1)

	for _, c := range []struct {
		vault domain.Vault
		want  chunk.SourceText
	}{
		{getFirst(), chunk.SourceText{Path: "library/first.pdf", Producer: "ocr", ContentHash: "hash-first"}},
		{getSecond(), chunk.SourceText{Path: "library/second.pdf", Producer: "ocr", ContentHash: "hash-second"}},
	} {
		found, err := db.ChunkFiles().GetRecognizedSources(t.Context(), c.vault.ID, "book")
		if err != nil {
			t.Fatal(err)
		}
		if len(found) != 1 || found[0] != c.want {
			t.Errorf("%s answers with %v, want only %v", c.vault.Name, found, c.want)
		}
	}
}

func saveSectionedBook(t *testing.T, db *DB, vault domain.Vault, path string) {
	t.Helper()
	ctx := t.Context()
	chunks := db.Chunks()

	if err := chunks.SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, ContentHash: "hash-" + path, Recipe: "pdf",
	}); err != nil {
		t.Fatal(err)
	}
	cutChunks := []chunk.Chunk{
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
			Small: []chunk.Chunk{{
				Start: 100, Length: 100,
				Text: "Madhavendra Puri was the disciple of Laksmipati. " +
					"Madhavendra Puri's disciples included Isvara Puri. " +
					"Madhavendra Puri is said to be. Madhavendra Puri again.",
			}},
		},
		{
			Start: 200, Length: 100, Location: "Alice Fenn",
			Opens: []string{"Alice Fenn"},
			Text:  "Alice Fenn met him at the far end of the site.",
			Small: []chunk.Chunk{{Start: 200, Length: 100, Text: "Alice Fenn met him."}},
		},
	}
	if err := chunks.ReplaceChunks(ctx, vault.ID, "book", path, cutChunks); err != nil {
		t.Fatal(err)
	}
}

func TestASectionIsFoundByItsName(t *testing.T) {
	// The word-search half returns the paragraph that mentions the name most often,
	// not the section about it.
	ctx := t.Context()
	db := openDB(t)
	saveSectionedBook(t, db, getFirst(), "library/chaitanya.pdf")
	queries := queriesOf(db)

	lexical, err := queries.FindLexical(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "Madhavendra Puri", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(lexical) == 0 {
		t.Fatal("the words half found nothing")
	}
	if lexical[0].Start == 0 {
		t.Fatal("the words half already answers with the section, and this proves nothing")
	}

	named, err := queries.GetNamedPassages(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "Madhavendra Puri", Kinds: nil, Limit: 10})
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
	// A small chunk positioned where a section begins is inside the chunk that
	// opens it. One section named twice produces two answers, and the second answer
	// is the same location.
	ctx := t.Context()
	db := openDB(t)
	saveSectionedBook(t, db, getFirst(), "library/chaitanya.pdf")

	named, err := db.ChunkSearch().GetNamedPassages(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "Madhavendra Puri", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 1 {
		t.Fatalf("%d sections came back, want the one", len(named))
	}
}

func TestASearchByNameStaysInsideItsVault(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	saveSectionedBook(t, db, getFirst(), "library/chaitanya.pdf")
	saveSectionedBook(t, db, getSecond(), "library/chaitanya.pdf")

	named, err := db.ChunkSearch().GetNamedPassages(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "Madhavendra Puri", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 1 {
		t.Fatalf("%d sections came back, want the one this vault holds", len(named))
	}
}

func TestASectionNamePreservesWithTheSourceItCameFrom(t *testing.T) {
	// A chunk id is reused by the next chunk that needs one, so a name
	ctx := t.Context()
	db := openDB(t)
	chunks := db.Chunks()

	if err := chunks.SaveSource(ctx, getFirst().ID, chunk.Source{
		Path: "library/gone.pdf", Kind: "book", Size: 1000, MTime: 1, ContentHash: "hash-gone", Recipe: "pdf",
	}); err != nil {
		t.Fatal(err)
	}
	if err := chunks.ReplaceChunks(ctx, getFirst().ID, "book", "library/gone.pdf", []chunk.Chunk{{
		Start: 0, Length: 100, Location: "Thermodynamics",
		Opens: []string{"Thermodynamics"},
		Text:  "Heat moves one way.",
		Small: []chunk.Chunk{{Start: 0, Length: 100, Text: "Heat moves one way."}},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := chunks.RemoveSources(ctx, getFirst().ID, "book", []string{"library/gone.pdf"}); err != nil {
		t.Fatal(err)
	}

	// The next book's chunks take the numbers the first one left, and this book
	// names no section of its own to write over them.
	if err := chunks.SaveSource(ctx, getFirst().ID, chunk.Source{
		Path: "library/next.pdf", Kind: "book", Size: 1000, MTime: 1, ContentHash: "hash-next", Recipe: "pdf",
	}); err != nil {
		t.Fatal(err)
	}
	if err := chunks.ReplaceChunks(ctx, getFirst().ID, "book", "library/next.pdf", []chunk.Chunk{{
		Start: 0, Length: 100, Location: "Whales",
		Text:  "A whale breathes air.",
		Small: []chunk.Chunk{{Start: 0, Length: 100, Text: "A whale breathes air."}},
	}}); err != nil {
		t.Fatal(err)
	}

	named, err := db.ChunkSearch().GetNamedPassages(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "Thermodynamics", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 0 {
		t.Errorf("a name of a source that is gone answers with %+v", named)
	}
}

func TestASectionSplitAwayIsNotFoundByItsName(t *testing.T) {
	// A chunk keeps its row through a split when its content is unchanged, so a
	// name is not removed with the chunk. A section that was removed but still
	// matches by name returns a passage opening where nothing begins.
	ctx := t.Context()
	db := openDB(t)
	saveSectionedBook(t, db, getFirst(), "library/chaitanya.pdf")

	// Cut again, and this time nothing opens a section.
	if err := db.Chunks().ReplaceChunks(ctx, getFirst().ID, "book", "library/chaitanya.pdf",
		[]chunk.Chunk{{
			Start: 0, Length: 100, Location: "Madhavendra Puri",
			Text:  "Madhavendra Puri appeared in the fourteenth century.",
			Small: []chunk.Chunk{{Start: 0, Length: 100, Text: "Madhavendra Puri appeared."}},
		}}); err != nil {
		t.Fatal(err)
	}

	named, err := db.ChunkSearch().GetNamedPassages(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "Madhavendra Puri", Kinds: nil, Limit: 10})
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
	db := openDB(t)

	if err := db.Sources().SaveExtraction(ctx, getFirst().ID, domain.SourceChunks{
		Source: domain.Source{
			FileStat: domain.FileStat{
				Path: "library/chaitanya.pdf", Kind: domain.KindFile, Size: 1000, ModTime: getWalked(),
			},
			ContentHash: "hash",
			Recipe:      "pdf",
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

	named, err := db.ChunkSearch().GetNamedPassages(ctx, getFirst().ID, sourcedomain.LexicalQuery{Text: "Madhavendra Puri", Kinds: nil, Limit: 10})
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

// Chunks of recently modified sources are returned ahead of older sources.
func TestRecentlyModifiedSourcesAreEmbeddedFirst(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)

	chunks := db.Chunks()
	if err := chunks.SaveSource(ctx, getFirst().ID, chunk.Source{
		Path: "library/old.epub", Kind: "book", Size: 1000, MTime: 100, ContentHash: "hash-old", Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	if err := chunks.ReplaceChunks(ctx, getFirst().ID, "book", "library/old.epub", []chunk.Chunk{{
		Start: 0, Length: 100, Text: "old book whole",
		Small: []chunk.Chunk{{Start: 0, Length: 50, Text: "old passage 1"}, {Start: 50, Length: 50, Text: "old passage 2"}},
	}}); err != nil {
		t.Fatal(err)
	}

	if err := chunks.SaveSource(ctx, getFirst().ID, chunk.Source{
		Path: "notes/recent.md", Kind: "note", Size: 500, MTime: 200, ContentHash: "hash-recent",
	}); err != nil {
		t.Fatal(err)
	}
	if err := chunks.ReplaceChunks(ctx, getFirst().ID, "note", "notes/recent.md", []chunk.Chunk{{
		Start: 0, Length: 100, Text: "recent note whole",
		Small: []chunk.Chunk{{Start: 0, Length: 50, Text: "recent passage 1"}, {Start: 50, Length: 50, Text: "recent passage 2"}},
	}}); err != nil {
		t.Fatal(err)
	}

	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(ctx, getFirst().ID, "model", chunk.Cursor{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) != 4 {
		t.Fatalf("expected 4 chunks unembedded, got %d", len(unembedded))
	}
	if unembedded[0].Path != "notes/recent.md" || unembedded[1].Path != "notes/recent.md" {
		t.Errorf("expected recent note chunks first, got: %s, %s", unembedded[0].Path, unembedded[1].Path)
	}
	if unembedded[2].Path != "library/old.epub" || unembedded[3].Path != "library/old.epub" {
		t.Errorf("expected old book chunks last, got: %s, %s", unembedded[2].Path, unembedded[3].Path)
	}
	if unembedded[0].Chunk >= unembedded[1].Chunk {
		t.Errorf("expected ascending chunk IDs within source, got %d and %d", unembedded[0].Chunk, unembedded[1].Chunk)
	}
	if unembedded[2].Chunk >= unembedded[3].Chunk {
		t.Errorf("expected ascending chunk IDs within source, got %d and %d", unembedded[2].Chunk, unembedded[3].Chunk)
	}

	firstPage, cursor1, err := db.Sources().GetUnembeddedChunks(ctx, getFirst().ID, getBenchModel(), "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstPage) != 2 || firstPage[0].Source != "notes/recent.md" || firstPage[1].Source != "notes/recent.md" {
		t.Fatalf("page 1 unexpected: %+v", firstPage)
	}

	secondPage, _, err := db.Sources().GetUnembeddedChunks(ctx, getFirst().ID, getBenchModel(), cursor1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPage) != 2 || secondPage[0].Source != "library/old.epub" || secondPage[1].Source != "library/old.epub" {
		t.Fatalf("page 2 unexpected: %+v", secondPage)
	}
}

func getTwoUnembedded(t *testing.T, db *DB) []chunk.Passage {
	t.Helper()
	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(t.Context(), getFirst().ID, "model", chunk.Cursor{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) != 2 {
		t.Fatalf("%d chunks need a vector, want two", len(unembedded))
	}
	return unembedded
}

func saveVectorPerChunk(t *testing.T, db *DB) {
	t.Helper()
	unembedded := getTwoUnembedded(t, db)
	made := make([]chunk.Vector, 0, len(unembedded))
	for _, p := range unembedded {
		made = append(made, chunk.Vector{
			Chunk: p.Chunk, Hash: hashOf(t, db, p.Chunk), Recipe: "model",
			Coarse: bits(0x00), Value: getPreciseBytes(getVectorForSeed(0x00)),
		})
	}
	if err := db.Chunks().SaveVectors(t.Context(), made); err != nil {
		t.Fatal(err)
	}
}
