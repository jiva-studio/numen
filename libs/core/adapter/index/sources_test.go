package index

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

var (
	// lastInstant is the last instant an index can stamp a source with.
	lastInstant = time.Unix(0, math.MaxInt64)
	// farFuture is past 2^62 nanoseconds since the epoch.
	farFuture = time.Date(2250, 1, 1, 0, 0, 0, 0, time.UTC)
	thisYear  = time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	longAgo   = time.Date(2020, 7, 1, 8, 30, 0, 0, time.UTC)
)

// saveThroughPort puts one book in through the source port, cut into a large
// chunk holding one small chunk for each text given.
func saveThroughPort(t *testing.T, db *DB, vault domain.Vault, path string, pos time.Time, texts ...string) {
	t.Helper()
	small := make([]domain.Chunk, 0, len(texts))
	for i, text := range texts {
		small = append(small, domain.Chunk{Start: i * 50, Length: 50, Text: text})
	}
	if err := db.Sources().SaveExtraction(t.Context(), vault.ID, domain.SourceChunks{
		Source: domain.Source{
			Fingerprint: domain.Fingerprint{Path: path, Kind: domain.KindBook, Size: 1000, ModTime: pos},
			Hash:        "hash-" + path,
			Recipe:      "epub",
		},
		Chunks: []domain.Chunk{{Start: 0, Length: 50 * len(texts), Text: strings.Join(texts, " "), Small: small}},
	}); err != nil {
		t.Fatal(err)
	}
}

// numberRowsPast2To62 makes every source and chunk written after it take a row
// number past 2^62, by holding a source and a chunk at that number in the
// second vault.
func numberRowsPast2To62(t *testing.T, db *DB) {
	t.Helper()
	const past = int64(1)<<62 + 1
	for _, statement := range []string{
		`INSERT INTO sources (id, vault_id, path, kind, size, modified_at)
		 VALUES (?, (SELECT id FROM vaults WHERE identifier = ?), 'library/numbering.epub', 'book', 1, 1)`,
		`INSERT INTO chunks (id, source_id, vault_id, start, length, hash)
		 VALUES (?1, ?1, (SELECT id FROM vaults WHERE identifier = ?2), 0, 1, 'aa')`,
	} {
		if _, err := db.write.ExecContext(t.Context(), statement, past, second.ID); err != nil {
			t.Fatal(err)
		}
	}
}

// getUnembeddedPassages is every chunk of a vault that owes a vector, asked in one page.
func getUnembeddedPassages(t *testing.T, db *DB, vault domain.Vault) []domain.Passage {
	t.Helper()
	owed, _, err := db.Sources().GetUnembeddedChunks(t.Context(), vault.ID, benchModel, "", 1000)
	if err != nil {
		t.Fatal(err)
	}
	return owed
}

// getUnembeddedPassagesPaged is every chunk of a vault that owes a vector, asked a page of the
// size given at a time, each page carrying on from the cursor the last handed
// back.
func getUnembeddedPassagesPaged(t *testing.T, db *DB, vault domain.Vault, limit int) []domain.Passage {
	t.Helper()
	var visited []domain.Passage
	cursor := embeddingport.ChunkCursor("")
	for range 100 {
		page, next, err := db.Sources().GetUnembeddedChunks(t.Context(), vault.ID, benchModel, cursor, limit)
		if err != nil {
			t.Fatalf("after cursor %q: %v", cursor, err)
		}
		if len(page) == 0 {
			return visited
		}
		visited = append(visited, page...)
		cursor = next
	}
	t.Fatalf("a walk in pages of %d did not end; it last stood at %q", limit, cursor)
	return nil
}

func getChunkIDs(passages []domain.Passage) []domain.ChunkID {
	out := make([]domain.ChunkID, 0, len(passages))
	for _, p := range passages {
		out = append(out, p.ChunkID)
	}
	return out
}

func getSourcePaths(passages []domain.Passage) []string {
	out := make([]string, 0, len(passages))
	for _, p := range passages {
		out = append(out, p.Source)
	}
	return out
}

// A walk over what owes a vector visits every owed chunk of its own vault once,
// newest source first, whatever the page size, whatever instant a source is
// stamped with and however high its rows are numbered.
func TestAWalkInPagesVisitsEveryOwedChunkOfItsVaultOnce(t *testing.T) {
	db := openDB(t)
	numberRowsPast2To62(t, db)
	saveThroughPort(t, db, first, "library/last.epub", lastInstant, "anchors weigh", "chains rattle")
	saveThroughPort(t, db, first, "library/far.epub", farFuture, "comets return", "orbits decay")
	saveThroughPort(t, db, first, "library/a.epub", thisYear, "bread rises", "yeast ferments", "crust browns")
	saveThroughPort(t, db, first, "library/b.epub", thisYear, "rivers bend", "deltas spread", "silt settles")
	saveThroughPort(t, db, first, "library/old.epub", longAgo, "glaciers carve", "moraines remain")
	saveThroughPort(t, db, second, "library/second.epub", thisYear, "violins tune", "cellos answer")

	owed := getUnembeddedPassages(t, db, first)
	want := []string{
		"library/last.epub", "library/last.epub",
		"library/far.epub", "library/far.epub",
		"library/b.epub", "library/b.epub", "library/b.epub",
		"library/a.epub", "library/a.epub", "library/a.epub",
		"library/old.epub", "library/old.epub",
	}
	if got := getSourcePaths(owed); !slices.Equal(got, want) {
		t.Fatalf("the first vault owes chunks of %v, want %v", got, want)
	}

	for _, limit := range []int{1, 2, 5} {
		visited := getUnembeddedPassagesPaged(t, db, first, limit)
		if got := getChunkIDs(visited); !slices.Equal(got, getChunkIDs(owed)) {
			t.Errorf("a walk in pages of %d visited %v, want %v", limit, got, getChunkIDs(owed))
		}
	}

	if got := getSourcePaths(getUnembeddedPassagesPaged(t, db, second, 1)); !slices.Equal(got, []string{"library/second.epub", "library/second.epub"}) {
		t.Errorf("a walk of the second vault visited chunks of %v", got)
	}
}

// A cursor of an earlier shape carries on no later than where it points: one
// number is a chunk at the start, and two are an instant and a chunk at it.
func TestACursorOfAnEarlierShapeCarriesOnNoLaterThanItPoints(t *testing.T) {
	db := openDB(t)
	saveThroughPort(t, db, first, "library/last.epub", lastInstant, "anchors weigh", "chains rattle")
	saveThroughPort(t, db, first, "library/later.epub", farFuture.AddDate(1, 0, 0), "tides turn", "moons wane")
	saveThroughPort(t, db, first, "library/far.epub", farFuture, "comets return", "orbits decay")

	// Every number is past 2^62.
	const row = "9000000000000000000"
	for _, item := range []struct {
		cursor embeddingport.ChunkCursor
		want   []string
	}{
		{row, []string{
			"library/last.epub", "library/last.epub",
			"library/later.epub", "library/later.epub",
			"library/far.epub", "library/far.epub",
		}},
		{embeddingport.ChunkCursor(strconv.FormatInt(farFuture.UnixNano(), 10) + ":" + row), []string{"library/far.epub", "library/far.epub"}},
	} {
		page, _, err := db.Sources().GetUnembeddedChunks(t.Context(), first.ID, benchModel, item.cursor, 10)
		if err != nil {
			t.Errorf("cursor %q: %v", item.cursor, err)
			continue
		}
		if got := getSourcePaths(page); !slices.Equal(got, item.want) {
			t.Errorf("cursor %q carried on with chunks of %v, want %v", item.cursor, got, item.want)
		}
	}
}

// A cursor this index did not hand out fails, and nothing is visited.
func TestACursorThisIndexDidNotHandOutIsRefused(t *testing.T) {
	db := openDB(t)
	saveThroughPort(t, db, first, "library/a.epub", thisYear, "bread rises", "yeast ferments")

	for _, cursor := range []embeddingport.ChunkCursor{"x", "x:1", "1:x", "x:1:2", "1:x:2", "1:2:x", "1:2:3:4"} {
		page, _, err := db.Sources().GetUnembeddedChunks(t.Context(), first.ID, benchModel, cursor, 10)
		if err == nil {
			t.Errorf("cursor %q was visited from, and gave %d chunks", cursor, len(page))
		}
	}
}

// A walk over an index that cannot be read is reported.
func TestAWalkOverAClosedIndexFails(t *testing.T) {
	db := openDB(t)
	saveThroughPort(t, db, first, "library/a.epub", thisYear, "bread rises", "yeast ferments")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if _, _, err := db.Sources().GetUnembeddedChunks(t.Context(), first.ID, benchModel, "", 10); err == nil {
		t.Error("a walk over a closed index reported nothing owed")
	}
}

// A source saved through the port reads back as it was handed over.
func TestASourceSavedThroughThePortReadsBackAsGiven(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	sources := db.Sources()

	if err := sources.SaveSource(ctx, first.ID, domain.Source{
		Fingerprint: domain.Fingerprint{Path: "library/scan.pdf", Kind: domain.KindBook, Size: 4321, ModTime: thisYear},
		Hash:        "hash-scan",
		Producer:    "ocr",
	}); err != nil {
		t.Fatal(err)
	}

	read, target, err := sources.GetSourceText(ctx, first.ID, "library/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !target {
		t.Fatal("the source saved is not held")
	}
	if read.Size != 4321 || !read.ModTime.Equal(thisYear) || read.Hash != "hash-scan" || read.Producer != "ocr" {
		t.Errorf("the source reads back as %+v", read)
	}
	books, err := sources.GetFingerprints(ctx, first.ID, domain.KindBook)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := books["library/scan.pdf"]; !found || len(books) != 1 {
		t.Errorf("the vault's books are %v, want the one saved", books)
	}
}

// A path the vault holds no source at is not held.
func TestAPathWithNoSourceIsNotHeld(t *testing.T) {
	db := openDB(t)
	saveThroughPort(t, db, second, "library/a.epub", thisYear, "violins tune")

	read, target, err := db.Sources().GetSourceText(t.Context(), first.ID, "library/a.epub")
	if err != nil {
		t.Fatal(err)
	}
	if target {
		t.Errorf("the first vault holds %+v, which only the second vault was given", read)
	}
}

// Vectors saved through the port settle what the chunks owed, and a vector for
// a chunk this index did not hand out fails.
func TestVectorsSavedThroughThePortSettleWhatWasOwed(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	saveThroughPort(t, db, first, "library/a.epub", thisYear, "bread rises", "yeast ferments")

	owed := getUnembeddedPassages(t, db, first)
	if len(owed) != 2 {
		t.Fatalf("%d chunks owe a vector, want 2", len(owed))
	}
	vectors := make([]embeddingport.Vector, 0, len(owed))
	for _, p := range owed {
		row, err := strconv.ParseInt(string(p.ChunkID), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		vectors = append(vectors, embeddingport.Vector{
			ChunkID: p.ChunkID, Hash: hashOf(t, db, row), Model: benchModel,
			Value: getPreciseBytes(getVectorForSeed(0x0f)), Coarse: bits(0x0f),
		})
	}
	if err := db.Sources().SaveVectors(ctx, vectors); err != nil {
		t.Fatal(err)
	}
	if remaining := getUnembeddedPassages(t, db, first); len(remaining) != 0 {
		t.Errorf("%d chunks still owe a vector", len(remaining))
	}

	stranger := embeddingport.Vector{ChunkID: "x", Hash: []byte{1}, Model: benchModel, Value: getPreciseBytes(getVectorForSeed(0x0f)), Coarse: bits(0x0f)}
	if err := db.Sources().SaveVectors(ctx, []embeddingport.Vector{stranger}); err == nil {
		t.Error("a vector for chunk \"x\" was saved")
	}
}

// The sources a producer made the text of are listed with the producer and the
// hash their recognition is kept under.
func TestRecognisedSourcesAreListedWithTheirProducer(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	if err := db.Sources().SaveSource(ctx, first.ID, domain.Source{
		Fingerprint: domain.Fingerprint{Path: "library/scan.pdf", Kind: domain.KindBook, Size: 4321, ModTime: thisYear},
		Hash:        "hash-scan",
		Producer:    "ocr",
	}); err != nil {
		t.Fatal(err)
	}

	found, err := db.Sources().GetRecognisedSources(ctx, first.ID, domain.KindBook)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Path != "library/scan.pdf" || found[0].Producer != "ocr" || found[0].Hash != "hash-scan" {
		t.Errorf("the recognised sources are %+v", found)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Sources().GetRecognisedSources(ctx, first.ID, domain.KindBook); err == nil {
		t.Error("listing over a closed index reported nothing recognised")
	}
}

// Reading a source out of an index that cannot be read is reported.
func TestReadingASourceOfAClosedIndexFails(t *testing.T) {
	db := openDB(t)
	saveThroughPort(t, db, first, "library/a.epub", thisYear, "bread rises")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if _, _, err := db.Sources().GetSourceText(t.Context(), first.ID, "library/a.epub"); err == nil {
		t.Error("reading a closed index reported the source as not held")
	}
}

// Removing sources from an index that cannot be written is reported.
func TestRemovingSourcesFromAClosedIndexFails(t *testing.T) {
	db := openDB(t)
	saveThroughPort(t, db, first, "library/a.epub", thisYear, "bread rises")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if err := db.Sources().RemoveSources(t.Context(), first.ID, domain.KindBook, []string{"library/a.epub"}); err == nil {
		t.Error("removing from a closed index reported success")
	}
}

// A cursor that fails gives no position to walk from, and one that is read
// gives the positions it names, the ones it leaves out standing at the end of
// the ordering.
func TestACursorIsReadIntoThePositionItNames(t *testing.T) {
	for _, c := range []struct {
		cursor               embeddingport.ChunkCursor
		mtime, source, chunk int64
		wantErr              bool
	}{
		{cursor: "", mtime: math.MaxInt64, source: math.MaxInt64, chunk: 0},
		{cursor: "7", mtime: math.MaxInt64, source: math.MaxInt64, chunk: 7},
		{cursor: "5:7", mtime: 5, source: math.MaxInt64, chunk: 7},
		{cursor: "5:6:7", mtime: 5, source: 6, chunk: 7},
		{cursor: "x", wantErr: true},
		{cursor: "x:1", wantErr: true},
		{cursor: "1:x", wantErr: true},
		{cursor: "x:1:2", wantErr: true},
		{cursor: "1:x:2", wantErr: true},
		{cursor: "1:2:x", wantErr: true},
		{cursor: "1:2:3:4", wantErr: true},
	} {
		got, err := parseCursor(c.cursor)
		mtime, source, chunk := got.MTime, got.Source, got.Chunk
		if c.wantErr != (err != nil) {
			t.Errorf("cursor %q: error %v, wantErr %v", c.cursor, err, c.wantErr)
		}
		if c.wantErr {
			c.mtime, c.source, c.chunk = 0, 0, 0
		}
		if mtime != c.mtime || source != c.source || chunk != c.chunk {
			t.Errorf("cursor %q read as %d, %d, %d, want %d, %d, %d", c.cursor, mtime, source, chunk, c.mtime, c.source, c.chunk)
		}
	}
}
