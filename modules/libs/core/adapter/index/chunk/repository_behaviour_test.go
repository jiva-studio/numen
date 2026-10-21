package chunk_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestTheSourceTimeColumnIsNanosecondsSinceTheEpoch(t *testing.T) {
	if got := chunk.ConvertNanosToTime(0); !got.IsZero() {
		t.Errorf("zero is a row that names no time, got %v", got)
	}
	if got := chunk.ConvertTimeToNanos(time.Time{}); got != 0 {
		t.Errorf("a time that was never set is %d, want 0", got)
	}
	for _, nanos := range []int64{1, -1, 1_700_000_000_123_456_789} {
		at := chunk.ConvertNanosToTime(nanos)
		if at.IsZero() {
			t.Errorf("%d nanoseconds is a time, got the zero one", nanos)
		}
		if got := at.UnixNano(); got != nanos {
			t.Errorf("%d nanoseconds reads back as %d", nanos, got)
		}
		if got := chunk.ConvertTimeToNanos(at); got != nanos {
			t.Errorf("%d nanoseconds written back as %d", nanos, got)
		}
	}
	if got := chunk.ConvertTimeToNanos(time.Unix(0, 0)); got != 0 {
		t.Errorf("the epoch is %d, want 0", got)
	}
	if got := chunk.ConvertTimeToNanos(time.Unix(3, 7)); got != 3_000_000_007 {
		t.Errorf("3s and 7ns is %d", got)
	}
}

func TestTheStatementsAreTheOnesTheRepositoryRuns(t *testing.T) {
	got := chunk.Statements()
	for _, name := range []string{"save_source", "insert_chunk", "keep_vector", "move_sources"} {
		if strings.TrimSpace(got[name]) == "" {
			t.Errorf("no statement is named %s", name)
		}
	}
}

func TestASavedSourceContainsWhatItWasGiven(t *testing.T) {
	s := newStore(t)
	err := s.db.Chunks().SaveSource(t.Context(), repoVaultA, chunk.Source{
		Path: "books/a.epub", Kind: "book", Size: 77, MTime: 1234, Hash: "h1", Recipe: "r1", Producer: "ocr",
	})
	if err != nil {
		t.Fatal(err)
	}

	var path, kind, hash, recipe, producer string
	var size, mtime int64
	if err := s.side.QueryRowContext(t.Context(),
		`SELECT path, kind, size, modified_at, hash, recipe, producer FROM sources`).
		Scan(&path, &kind, &size, &mtime, &hash, &recipe, &producer); err != nil {
		t.Fatal(err)
	}
	if path != "books/a.epub" || kind != "book" || size != 77 || mtime != 1234 ||
		hash != "h1" || recipe != "r1" || producer != "ocr" {
		t.Errorf("the row holds %q %q %d %d %q %q %q", path, kind, size, mtime, hash, recipe, producer)
	}
	if got := s.paths(repoVaultB); len(got) != 0 {
		t.Errorf("the other vault holds %v", got)
	}
}

func TestASourceSavedAgainKeepsItsRowAndConsumesTheNewValues(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	first := chunk.Source{Path: "a.epub", Kind: "book", Size: 1, MTime: 10, Hash: "h1", Recipe: "r1", Producer: "p1"}
	if err := repo.SaveSource(t.Context(), repoVaultA, first); err != nil {
		t.Fatal(err)
	}
	id := s.ints(`SELECT id FROM sources`)

	if err := repo.SaveSource(t.Context(), repoVaultA, chunk.Source{Path: "a.epub", Kind: "book", Size: 2, MTime: 20}); err != nil {
		t.Fatal(err)
	}

	if got := s.ints(`SELECT id FROM sources`); !slices.Equal(got, id) {
		t.Errorf("the rows are %v, were %v", got, id)
	}
	if n := s.count(`SELECT count(*) FROM sources WHERE size = 2 AND modified_at = 20`); n != 1 {
		t.Errorf("the new size and time were not taken")
	}
	if n := s.count(`SELECT count(*) FROM sources WHERE hash IS NULL AND recipe IS NULL AND producer IS NULL`); n != 1 {
		t.Errorf("an empty hash, recipe and producer are recorded as values, not as nothing")
	}
}

func TestAVaultTheIndexDoesNotKnowContainsNothingAndIsSavedNothing(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	const stranger domain.VaultID = "01STRANGER"
	src := chunk.Source{Path: "a.epub", Kind: "book", Size: 1, MTime: 1}

	if err := repo.SaveSource(t.Context(), stranger, src); err == nil {
		t.Error("a source was saved into a vault the index does not know")
	}
	if err := repo.SaveExtraction(t.Context(), stranger, src, []chunk.Chunk{{Text: "x"}}); err == nil {
		t.Error("an extraction was saved into a vault the index does not know")
	}
	if err := repo.ReplaceChunks(t.Context(), stranger, "book", "a.epub", nil); err == nil {
		t.Error("chunks were replaced in a vault the index does not know")
	}
	if err := repo.RemoveSources(t.Context(), stranger, "book", []string{"a.epub"}); err == nil {
		t.Error("sources were removed from a vault the index does not know")
	}
	if err := repo.RemoveSources(t.Context(), stranger, "book", nil); err != nil {
		t.Errorf("removing nothing from an unknown vault is %v", err)
	}
	if err := repo.MoveSources(t.Context(), stranger, "a", "b"); err != nil {
		t.Errorf("moving in a vault the index does not know is %v, and it holds nothing to move", err)
	}
	if n := s.count(`SELECT count(*) FROM sources`); n != 0 {
		t.Errorf("%d sources were written", n)
	}
}

func TestAnIndexThatCannotNameTheVaultReturnsWhyOnEveryWrite(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	s.exec(`DROP TABLE vaults`)
	src := chunk.Source{Path: "a.epub", Kind: "book", Size: 1, MTime: 1}

	for name, run := range map[string]func() error{
		"SaveSource":     func() error { return repo.SaveSource(t.Context(), repoVaultA, src) },
		"SaveExtraction": func() error { return repo.SaveExtraction(t.Context(), repoVaultA, src, nil) },
		"ReplaceChunks":  func() error { return repo.ReplaceChunks(t.Context(), repoVaultA, "book", "a.epub", nil) },
		"RemoveSources":  func() error { return repo.RemoveSources(t.Context(), repoVaultA, "book", []string{"a.epub"}) },
		"MoveSources":    func() error { return repo.MoveSources(t.Context(), repoVaultA, "a", "b") },
	} {
		t.Run(name, func(t *testing.T) {
			err := run()
			wantSQLiteError(t, err, "no such table")
			if strings.Contains(err.Error(), "vault not in the index") {
				t.Errorf("a broken index is reported as an unknown vault: %v", err)
			}
		})
	}
}

func TestAWriteWhoseContextIsMissingReturnsItCouldNotBegin(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	src := chunk.Source{Path: "a.epub", Kind: "book", Size: 1, MTime: 1}
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{{Start: 0, Length: 5, Text: "x"}})
	row := s.rowsOf(repoVaultA, "a.epub")[0]
	vector := s.vectorFor(row, "m", 1)

	for name, run := range map[string]func() error{
		"SaveSource":     func() error { return repo.SaveSource(canceled(t), repoVaultA, src) },
		"SaveExtraction": func() error { return repo.SaveExtraction(canceled(t), repoVaultA, src, nil) },
		"ReplaceChunks":  func() error { return repo.ReplaceChunks(canceled(t), repoVaultA, "book", "a.epub", nil) },
		"SaveVectors":    func() error { return repo.SaveVectors(canceled(t), []chunk.Vector{vector}) },
		"RemoveSources":  func() error { return repo.RemoveSources(canceled(t), repoVaultA, "book", []string{"a.epub"}) },
		"MoveSources":    func() error { return repo.MoveSources(canceled(t), repoVaultA, "a", "b") },
	} {
		t.Run(name, func(t *testing.T) {
			err := run()
			if err == nil {
				t.Fatal("a write with no context left succeeded")
			}
			if !strings.HasPrefix(err.Error(), "begin: ") {
				t.Errorf("the error %q does not say it could not begin", err)
			}
			if !isCanceled(err) {
				t.Errorf("the error %q no longer says why", err)
			}
		})
	}
	if got := s.rowsOf(repoVaultA, "a.epub"); len(got) != 1 {
		t.Errorf("the source's chunks are %v", got)
	}
}

func TestAnExtractionIsASourceAndItsChunksTogether(t *testing.T) {
	s := newStore(t)
	chunks := []chunk.Chunk{{Start: 0, Length: 20, Location: "ch 1", Text: "whole", Small: []chunk.Chunk{
		{Start: 0, Length: 10, Text: "left"}, {Start: 10, Length: 10, Text: "right"},
	}}}
	src := chunk.Source{Path: "a.epub", Kind: "book", Size: 5, MTime: 9, Hash: "h", Recipe: "r", Producer: "p"}

	if err := s.db.Chunks().SaveExtraction(t.Context(), repoVaultA, src, chunks); err != nil {
		t.Fatal(err)
	}

	if n := s.count(`SELECT count(*) FROM sources WHERE path = 'a.epub' AND size = 5 AND modified_at = 9
		AND hash = 'h' AND recipe = 'r' AND producer = 'p'`); n != 1 {
		t.Error("the source was not recorded as given")
	}
	if got := s.rowsOf(repoVaultA, "a.epub"); len(got) != 3 {
		t.Errorf("the extraction holds %d chunks, want 3", len(got))
	}
}

func TestAnExtractionThatFailsPreservesNeitherTheSourceNorTheChunks(t *testing.T) {
	s := newStore(t)
	s.exec(`CREATE TRIGGER refuse BEFORE INSERT ON chunks BEGIN SELECT RAISE(ABORT, 'boom'); END`)

	err := s.db.Chunks().SaveExtraction(t.Context(), repoVaultA,
		chunk.Source{Path: "a.epub", Kind: "book", Size: 5, MTime: 9}, []chunk.Chunk{{Start: 0, Length: 1, Text: "x"}})

	wantSQLiteError(t, err, "boom")
	if got := s.paths(repoVaultA); len(got) != 0 {
		t.Errorf("the source %v was kept", got)
	}
}

func TestChunksReplacedForASourceTheIndexDoesNotHoldSayWhich(t *testing.T) {
	s := newStore(t)
	s.cut(repoVaultA, "book", "a.epub", nil)

	err := s.db.Chunks().ReplaceChunks(t.Context(), repoVaultA, "note", "a.epub", nil)
	if err == nil || err.Error() != "a.epub is not a note the index holds" {
		t.Errorf("a source of another kind: %v", err)
	}
	err = s.db.Chunks().ReplaceChunks(t.Context(), repoVaultB, "book", "a.epub", nil)
	if err == nil || err.Error() != "a.epub is not a book the index holds" {
		t.Errorf("a source of another vault: %v", err)
	}
	err = s.db.Chunks().ReplaceChunks(t.Context(), repoVaultA, "book", "b.epub", nil)
	if err == nil || err.Error() != "b.epub is not a book the index holds" {
		t.Errorf("a source that is not there: %v", err)
	}
}

func TestTheSplitOfASourceContainsALargeChunkAndTheSmallOnesInsideIt(t *testing.T) {
	s := newStore(t)
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{
		{Start: 0, Length: 20, Location: "chapter 1", Text: "whole one", Small: []chunk.Chunk{
			{Start: 0, Length: 10, Text: "alpha"}, {Start: 10, Length: 10, Location: "p 2", Text: "beta"},
		}},
		{Start: 20, Length: 5, Text: "lone"},
	})

	rows := s.rowsOf(repoVaultA, "a.epub")
	if len(rows) != 4 {
		t.Fatalf("the cut holds %d chunks, want 4", len(rows))
	}
	large, alpha, beta, lone := rows[0], rows[1], rows[2], rows[3]
	for row, want := range map[int64]string{large: "0 20 chapter 1", alpha: "0 10 -", beta: "10 10 p 2", lone: "20 5 -"} {
		got := s.strings(`SELECT start || ' ' || length || ' ' || COALESCE(location, '-') FROM chunks WHERE id = ?`, row)
		if len(got) != 1 || got[0] != want {
			t.Errorf("chunk %d is %v, want %q", row, got, want)
		}
	}
	for row, parent := range map[int64]int64{alpha: large, beta: large} {
		if n := s.count(`SELECT count(*) FROM chunks WHERE id = ? AND parent_id = ?`, row, parent); n != 1 {
			t.Errorf("chunk %d does not sit inside chunk %d", row, parent)
		}
	}
	if n := s.count(`SELECT count(*) FROM chunks WHERE parent_id IS NULL`); n != 2 {
		t.Errorf("%d chunks are large, want 2", n)
	}
	// The words are indexed on the chunks nothing is inside.
	if got := s.ints(`SELECT rowid FROM chunks_fts ORDER BY rowid`); !slices.Equal(got, []int64{alpha, beta, lone}) {
		t.Errorf("the words are indexed on %v, want %v", got, []int64{alpha, beta, lone})
	}
	hash := s.strings(`SELECT hash FROM chunks WHERE id = ?`, alpha)
	if len(hash) != 1 || len(hash[0]) != 64 {
		t.Errorf("a chunk is addressed by %v", hash)
	}
	other := s.strings(`SELECT hash FROM chunks WHERE id = ?`, beta)
	if hash[0] == other[0] {
		t.Error("two texts are addressed by one hash")
	}
}

func TestASplitAgainKeepsTheRowOfEveryChunkWhoseTextSurvivesAndDropsTheRest(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{{
		Start: 0, Length: 20, Text: "whole one", Small: []chunk.Chunk{
			{Start: 0, Length: 10, Text: "alpha"}, {Start: 10, Length: 10, Text: "beta"},
		},
	}})
	before := s.rowsOf(repoVaultA, "a.epub")
	oldLarge, alpha, beta := before[0], before[1], before[2]
	if err := repo.SaveVectors(t.Context(), []chunk.Vector{s.vectorFor(alpha, "m", 1), s.vectorFor(beta, "m", 2)}); err != nil {
		t.Fatal(err)
	}
	betaHash := s.strings(`SELECT hash FROM chunks WHERE id = ?`, beta)[0]

	if err := repo.ReplaceChunks(t.Context(), repoVaultA, "book", "a.epub", []chunk.Chunk{{
		Start: 0, Length: 30, Location: "moved", Text: "whole two", Small: []chunk.Chunk{
			{Start: 5, Length: 7, Location: "new place", Text: "alpha"}, {Start: 12, Length: 8, Text: "gamma"},
		},
	}}); err != nil {
		t.Fatal(err)
	}

	after := s.rowsOf(repoVaultA, "a.epub")
	if len(after) != 3 {
		t.Fatalf("the source holds %v", after)
	}
	if !slices.Contains(after, alpha) {
		t.Fatalf("the chunk whose text survived lost its row: %v, was %v", after, before)
	}
	if slices.Contains(after, oldLarge) || slices.Contains(after, beta) {
		t.Errorf("a chunk whose text is gone kept its row: %v", after)
	}
	newLarge := s.ints(`SELECT id FROM chunks WHERE parent_id IS NULL`)
	if len(newLarge) != 1 {
		t.Fatalf("the source holds %v large chunks", newLarge)
	}
	if got := s.strings(`SELECT start || ' ' || length || ' ' || location || ' ' || parent_id FROM chunks WHERE id = ?`, alpha); len(got) != 1 ||
		got[0] != fmt.Sprintf("5 7 new place %d", newLarge[0]) {
		t.Errorf("the kept chunk is %v, want it moved inside the new large chunk %d", got, newLarge[0])
	}
	if n := s.count(`SELECT count(*) FROM chunks WHERE location = 'moved' AND id = ?`, newLarge[0]); n != 1 {
		t.Error("the new large chunk lost its location")
	}
	// What the kept chunk was given stays given; what went takes its rows with it.
	if n := s.count(`SELECT count(*) FROM chunks_vec WHERE chunk_id = ?`, alpha); n != 1 {
		t.Error("the kept chunk lost its coarse row")
	}
	if n := s.count(`SELECT count(*) FROM chunks_vec WHERE chunk_id IN (?, ?)`, beta, oldLarge); n != 0 {
		t.Error("a chunk that went left its coarse row")
	}
	if n := s.count(`SELECT count(*) FROM chunks_fts WHERE rowid IN (?, ?)`, beta, oldLarge); n != 0 {
		t.Error("a chunk that went left its words")
	}
	if n := s.count(`SELECT count(*) FROM chunks_fts WHERE rowid = ?`, alpha); n != 1 {
		t.Error("the kept chunk lost its words")
	}
	if n := s.count(`SELECT count(*) FROM vectors WHERE hex(hash) = upper(?)`, betaHash); n != 0 {
		t.Error("the vector bought for text that is gone was kept")
	}
	if n := s.count(`SELECT count(*) FROM vectors`); n != 1 {
		t.Errorf("%d vectors are held, want the kept chunk's", n)
	}
}

func TestAVectorForTextAnotherChunkStillContainsOutlivesTheChunkItWasBoughtFor(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{{Start: 0, Length: 5, Text: "shared words"}})
	s.cut(repoVaultA, "book", "b.epub", []chunk.Chunk{{Start: 0, Length: 5, Text: "shared words"}})
	row := s.rowsOf(repoVaultA, "a.epub")[0]
	if err := repo.SaveVectors(t.Context(), []chunk.Vector{s.vectorFor(row, "m", 1)}); err != nil {
		t.Fatal(err)
	}

	if err := repo.ReplaceChunks(t.Context(), repoVaultA, "book", "a.epub", []chunk.Chunk{{Start: 0, Length: 5, Text: "other words"}}); err != nil {
		t.Fatal(err)
	}

	if n := s.count(`SELECT count(*) FROM vectors`); n != 1 {
		t.Errorf("%d vectors held, want the one the other source still needs", n)
	}
}

func TestAChunkIsCachedOnlyBySameTextAtTheSameSize(t *testing.T) {
	s := newStore(t)
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{{Start: 0, Length: 5, Text: "same"}})
	was := s.rowsOf(repoVaultA, "a.epub")[0]

	if err := s.db.Chunks().ReplaceChunks(t.Context(), repoVaultA, "book", "a.epub", []chunk.Chunk{{
		Start: 0, Length: 5, Text: "outer", Small: []chunk.Chunk{{Start: 0, Length: 5, Text: "same"}},
	}}); err != nil {
		t.Fatal(err)
	}

	if got := s.rowsOf(repoVaultA, "a.epub"); slices.Contains(got, was) || len(got) != 2 {
		t.Errorf("a large chunk's row was handed to a small one with the same text: %v, was %d", got, was)
	}
}

func TestATextThatOccursTwiceIsTwoRowsAndEachIsCachedOnce(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{{Start: 0, Length: 5, Text: "echo"}, {Start: 5, Length: 5, Text: "echo"}})
	was := s.rowsOf(repoVaultA, "a.epub")
	if len(was) != 2 {
		t.Fatalf("two chunks hold %v", was)
	}

	if err := repo.ReplaceChunks(t.Context(), repoVaultA, "book", "a.epub", []chunk.Chunk{
		{Start: 0, Length: 5, Text: "echo"}, {Start: 5, Length: 5, Text: "echo"}, {Start: 10, Length: 5, Text: "echo"},
	}); err != nil {
		t.Fatal(err)
	}
	three := s.rowsOf(repoVaultA, "a.epub")
	if len(three) != 3 || !slices.Contains(three, was[0]) || !slices.Contains(three, was[1]) {
		t.Errorf("the rows are %v, want %v kept and one added", three, was)
	}

	if err := repo.ReplaceChunks(t.Context(), repoVaultA, "book", "a.epub", []chunk.Chunk{{Start: 0, Length: 5, Text: "echo"}}); err != nil {
		t.Fatal(err)
	}
	if one := s.rowsOf(repoVaultA, "a.epub"); len(one) != 1 {
		t.Errorf("one chunk left %v", one)
	}
	if n := s.count(`SELECT count(*) FROM chunks_fts`); n != 1 {
		t.Errorf("%d rows of words are held for one chunk", n)
	}
}

func TestTheSectionsAChunkOpensAreIndexedByName(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{
		{Start: 0, Length: 5, Text: "first", Opens: []string{"Beginnings", "Prologue"}},
		{Start: 5, Length: 5, Text: "second"},
	})
	rows := s.rowsOf(repoVaultA, "a.epub")
	first, second := rows[0], rows[1]

	match := func(name string) []int64 {
		return s.ints(`SELECT rowid FROM sections_fts WHERE sections_fts MATCH ?`, name)
	}
	if got := match("Beginnings"); !slices.Equal(got, []int64{first}) {
		t.Errorf("Beginnings names %v, want %d", got, first)
	}
	if got := match("Prologue"); !slices.Equal(got, []int64{first}) {
		t.Errorf("Prologue names %v, want %d", got, first)
	}
	if n := s.count(`SELECT count(*) FROM sections_fts WHERE rowid = ?`, second); n != 0 {
		t.Error("a chunk that opens nothing is indexed by name")
	}

	// The same text cut again keeps its row and takes the names it opens now.
	if err := repo.ReplaceChunks(t.Context(), repoVaultA, "book", "a.epub", []chunk.Chunk{
		{Start: 0, Length: 5, Text: "first", Opens: []string{"Renamed"}},
		{Start: 5, Length: 5, Text: "second", Opens: []string{"Epilogue"}},
	}); err != nil {
		t.Fatal(err)
	}
	if got := match("Beginnings"); len(got) != 0 {
		t.Errorf("a name the chunk no longer opens still finds %v", got)
	}
	if got := match("Renamed"); !slices.Equal(got, []int64{first}) {
		t.Errorf("Renamed names %v, want the kept chunk %d", got, first)
	}
	if got := match("Epilogue"); !slices.Equal(got, []int64{second}) {
		t.Errorf("Epilogue names %v, want the kept chunk %d", got, second)
	}

	if err := repo.ReplaceChunks(t.Context(), repoVaultA, "book", "a.epub", []chunk.Chunk{
		{Start: 0, Length: 5, Text: "first"}, {Start: 5, Length: 5, Text: "second"},
	}); err != nil {
		t.Fatal(err)
	}
	if n := s.count(`SELECT count(*) FROM sections_fts`); n != 0 {
		t.Errorf("%d names remain for chunks that open nothing", n)
	}
}

func TestASplitKeepsAnotherSourceOfTheSameTextApart(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{{Start: 0, Length: 5, Text: "same"}})
	s.cut(repoVaultA, "book", "b.epub", []chunk.Chunk{{Start: 0, Length: 5, Text: "same"}})
	other := s.rowsOf(repoVaultA, "b.epub")

	if err := repo.ReplaceChunks(t.Context(), repoVaultA, "book", "a.epub", nil); err != nil {
		t.Fatal(err)
	}

	if got := s.rowsOf(repoVaultA, "a.epub"); len(got) != 0 {
		t.Errorf("the source still holds %v", got)
	}
	if got := s.rowsOf(repoVaultA, "b.epub"); !slices.Equal(got, other) {
		t.Errorf("the other source holds %v, was %v", got, other)
	}
}

func TestVectorsAreSavedForEveryChunkTheyNameAndReplacedWhenSavedAgain(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{{Start: 0, Length: 5, Text: "one"}, {Start: 5, Length: 5, Text: "two"}})
	rows := s.rowsOf(repoVaultA, "a.epub")

	if err := repo.SaveVectors(t.Context(), nil); err != nil {
		t.Fatalf("no vectors: %v", err)
	}
	if n := s.count(`SELECT count(*) FROM vectors`); n != 0 {
		t.Fatalf("%d vectors from none", n)
	}

	removed := withHash(9999, []byte("no chunk holds this"), "m", 7)
	if err := repo.SaveVectors(t.Context(), []chunk.Vector{s.vectorFor(rows[0], "m", 1), removed, s.vectorFor(rows[1], "m", 2)}); err != nil {
		t.Fatal(err)
	}
	if got := s.ints(`SELECT chunk_id FROM chunks_vec ORDER BY chunk_id`); !slices.Equal(got, rows) {
		t.Errorf("the coarse index names %v, want %v", got, rows)
	}
	if n := s.count(`SELECT count(*) FROM vectors`); n != 3 {
		t.Errorf("%d vectors are kept, want the one a gone chunk was bought too", n)
	}
	if n := s.count(`SELECT count(*) FROM chunks_vec WHERE vault_id = (SELECT id FROM vaults WHERE identifier = ?)`, string(repoVaultA)); n != 2 {
		t.Errorf("%d coarse rows carry the chunk's vault", n)
	}

	if err := repo.SaveVectors(t.Context(), []chunk.Vector{s.vectorFor(rows[0], "m", 9)}); err != nil {
		t.Fatal(err)
	}
	if n := s.count(`SELECT count(*) FROM vectors`); n != 3 {
		t.Errorf("%d vectors held after one was written again", n)
	}
	var embedding []byte
	if err := s.side.QueryRowContext(t.Context(),
		`SELECT embedding FROM vectors WHERE hash = (SELECT unhex(hash) FROM chunks WHERE id = ?)`, rows[0]).Scan(&embedding); err != nil {
		t.Fatal(err)
	}
	if len(embedding) != 1024 || embedding[0] != 9 {
		t.Errorf("the vector written again was not taken: %d bytes starting %d", len(embedding), embedding[0])
	}
}

func TestVectorsGoInBatchesEachCommittedOnItsOwn(t *testing.T) {
	for _, size := range []int{1, 2, 3, 4, 7} {
		t.Run(fmt.Sprintf("batch of %d", size), func(t *testing.T) {
			s := newStore(t)
			was := chunk.VectorBatchSize
			chunk.VectorBatchSize = size
			t.Cleanup(func() { chunk.VectorBatchSize = was })
			var trimmed []chunk.Chunk
			for i := range 6 {
				trimmed = append(trimmed, chunk.Chunk{Start: i * 5, Length: 5, Text: fmt.Sprintf("chunk %d", i)})
			}
			s.cut(repoVaultA, "book", "a.epub", trimmed)
			var vectors []chunk.Vector
			for _, row := range s.rowsOf(repoVaultA, "a.epub") {
				vectors = append(vectors, s.vectorFor(row, "m", 1))
			}

			if err := s.db.Chunks().SaveVectors(t.Context(), vectors); err != nil {
				t.Fatal(err)
			}

			if n := s.count(`SELECT count(*) FROM vectors`); n != 6 {
				t.Errorf("%d vectors kept, want 6", n)
			}
			if n := s.count(`SELECT count(*) FROM chunks_vec`); n != 6 {
				t.Errorf("%d coarse rows kept, want 6", n)
			}
		})
	}
}

func TestAVectorThatFailsPreservesTheBatchesBeforeItSaved(t *testing.T) {
	s := newStore(t)
	was := chunk.VectorBatchSize
	chunk.VectorBatchSize = 2
	t.Cleanup(func() { chunk.VectorBatchSize = was })
	var trimmed []chunk.Chunk
	for i := range 5 {
		trimmed = append(trimmed, chunk.Chunk{Start: i * 5, Length: 5, Text: fmt.Sprintf("chunk %d", i)})
	}
	s.cut(repoVaultA, "book", "a.epub", trimmed)
	rows := s.rowsOf(repoVaultA, "a.epub")
	s.exec(`CREATE TRIGGER refuse BEFORE INSERT ON vectors WHEN NEW.recipe = 'poison' BEGIN SELECT RAISE(ABORT, 'boom'); END`)
	vectors := []chunk.Vector{
		s.vectorFor(rows[0], "m", 1), s.vectorFor(rows[1], "m", 1),
		s.vectorFor(rows[2], "m", 1), s.vectorFor(rows[3], "poison", 1), s.vectorFor(rows[4], "m", 1),
	}

	err := s.db.Chunks().SaveVectors(t.Context(), vectors)

	wantSQLiteError(t, err, "keep_vector: ")
	if n := s.count(`SELECT count(*) FROM vectors`); n != 2 {
		t.Errorf("%d vectors kept, want the first batch's two", n)
	}
	if n := s.count(`SELECT count(*) FROM chunks_vec`); n != 2 {
		t.Errorf("%d coarse rows kept, want the first batch's two", n)
	}
}

func TestAThousandVectorsAreOneBatchAndTheThousandAndFirstIsAnother(t *testing.T) {
	s := newStore(t)
	var trimmed []chunk.Chunk
	for i := range 1001 {
		trimmed = append(trimmed, chunk.Chunk{Start: i, Length: 1, Text: fmt.Sprintf("chunk %d", i)})
	}
	s.cut(repoVaultA, "book", "a.epub", trimmed)
	rows := s.rowsOf(repoVaultA, "a.epub")
	s.exec(`CREATE TRIGGER refuse BEFORE INSERT ON vectors WHEN NEW.recipe = 'poison' BEGIN SELECT RAISE(ABORT, 'boom'); END`)
	vectors := make([]chunk.Vector, 0, len(rows))
	for i, row := range rows {
		recipe := "m"
		if i == len(rows)-1 {
			recipe = "poison"
		}
		vectors = append(vectors, s.vectorFor(row, recipe, 1))
	}

	err := s.db.Chunks().SaveVectors(t.Context(), vectors)

	wantSQLiteError(t, err, "boom")
	if n := s.count(`SELECT count(*) FROM vectors`); n != 1000 {
		t.Errorf("%d vectors kept, want the first thousand", n)
	}
}

func TestRemovedSourcesTakeEverythingMadeFromThem(t *testing.T) {
	s := newStore(t)
	repo := s.db.Chunks()
	for _, path := range []string{"a.epub", "b.epub", "c.epub"} {
		s.cut(repoVaultA, "book", path, []chunk.Chunk{{Start: 0, Length: 9, Text: "whole " + path, Opens: []string{"Name " + path}, Small: []chunk.Chunk{
			{Start: 0, Length: 4, Text: "left " + path},
		}}})
	}
	s.cut(repoVaultB, "book", "a.epub", []chunk.Chunk{{Start: 0, Length: 4, Text: "elsewhere"}})
	s.cut(repoVaultA, "note", "n.md", []chunk.Chunk{{Start: 0, Length: 4, Text: "a note"}})
	var vectors []chunk.Vector
	for _, row := range s.rowsOf(repoVaultA, "a.epub") {
		vectors = append(vectors, s.vectorFor(row, "m", 1))
	}
	if err := repo.SaveVectors(t.Context(), vectors); err != nil {
		t.Fatal(err)
	}
	keptB := s.rowsOf(repoVaultA, "b.epub")
	keptOther := s.rowsOf(repoVaultB, "a.epub")

	if err := repo.RemoveSources(t.Context(), repoVaultA, "book", []string{"missing.epub", "a.epub", "n.md", "c.epub"}); err != nil {
		t.Fatal(err)
	}

	if got := s.paths(repoVaultA); !slices.Equal(got, []string{"b.epub", "n.md"}) {
		t.Errorf("the vault holds %v, want only what was not named, or named of another kind", got)
	}
	if got := s.paths(repoVaultB); !slices.Equal(got, []string{"a.epub"}) {
		t.Errorf("the other vault holds %v", got)
	}
	if got := s.rowsOf(repoVaultA, "b.epub"); !slices.Equal(got, keptB) {
		t.Errorf("a source that stayed holds %v, was %v", got, keptB)
	}
	if got := s.rowsOf(repoVaultB, "a.epub"); !slices.Equal(got, keptOther) {
		t.Errorf("the other vault's source holds %v, was %v", got, keptOther)
	}
	if n := s.count(`SELECT count(*) FROM chunks`); n != 2+1+1 {
		t.Errorf("%d chunks are held", n)
	}
	for _, table := range []string{"chunks_fts", "chunks_vec", "sections_fts"} {
		rowCount := s.count(`SELECT count(*) FROM ` + table)
		want := map[string]int{"chunks_fts": 3, "chunks_vec": 0, "sections_fts": 1}[table]
		if rowCount != want {
			t.Errorf("%s holds %d rows, want %d", table, rowCount, want)
		}
	}
	if n := s.count(`SELECT count(*) FROM vectors`); n != 2 {
		t.Errorf("%d vectors are held: a vector outlives the source it was bought for", n)
	}
}

func TestRemovingNoSourcesChangesNothing(t *testing.T) {
	s := newStore(t)
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{{Start: 0, Length: 4, Text: "x"}})

	if err := s.db.Chunks().RemoveSources(t.Context(), repoVaultA, "book", nil); err != nil {
		t.Fatal(err)
	}

	if got := s.paths(repoVaultA); !slices.Equal(got, []string{"a.epub"}) {
		t.Errorf("the vault holds %v", got)
	}
}
