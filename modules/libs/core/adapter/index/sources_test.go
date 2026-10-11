package index

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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
func saveThroughPort(t *testing.T, db *DB, vault domain.Vault, path string, at time.Time, texts ...string) {
	t.Helper()
	small := make([]domain.Chunk, 0, len(texts))
	for i, text := range texts {
		small = append(small, domain.Chunk{Start: i * 50, Length: 50, Text: text})
	}
	if err := db.Sources().SaveExtraction(t.Context(), vault.ID, domain.SourceChunks{
		Source: domain.Source{
			Fingerprint: domain.Fingerprint{Path: path, Kind: domain.KindBook, Size: 1000, ModTime: at},
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

// getOwed is every chunk of a vault that owes a vector, asked in one page.
func getOwed(t *testing.T, db *DB, vault domain.Vault) []domain.Passage {
	t.Helper()
	owed, _, err := db.Sources().GetUnembeddedChunks(t.Context(), vault.ID, benchModel, "", 1000)
	if err != nil {
		t.Fatal(err)
	}
	return owed
}

// walkOwed is every chunk of a vault that owes a vector, asked a page of the
// size given at a time, each page carrying on from the cursor the last handed
// back.
func walkOwed(t *testing.T, db *DB, vault domain.Vault, limit int) []domain.Passage {
	t.Helper()
	var visited []domain.Passage
	cursor := port.ChunkCursor("")
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

	owed := getOwed(t, db, first)
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
		visited := walkOwed(t, db, first, limit)
		if got := getChunkIDs(visited); !slices.Equal(got, getChunkIDs(owed)) {
			t.Errorf("a walk in pages of %d visited %v, want %v", limit, got, getChunkIDs(owed))
		}
	}

	if got := getSourcePaths(walkOwed(t, db, second, 1)); !slices.Equal(got, []string{"library/second.epub", "library/second.epub"}) {
		t.Errorf("a walk of the second vault visited chunks of %v", got)
	}
}

// A cursor of an earlier shape carries on no later than where it points: one
// number is a chunk at the start, and two are an instant and a chunk at it.
func TestACursorOfAnEarlierShapeCarriesOnNoLaterThanItPoints(t *testing.T) {
	db := openDB(t)
	saveThroughPort(t, db, first, "library/a.epub", thisYear, "bread rises", "yeast ferments")
	saveThroughPort(t, db, first, "library/old.epub", longAgo, "glaciers carve", "moraines remain")

	for _, at := range []struct {
		cursor port.ChunkCursor
		want   []string
	}{
		{"1", []string{"library/a.epub", "library/a.epub", "library/old.epub", "library/old.epub"}},
		{port.ChunkCursor(strconv.FormatInt(longAgo.UnixNano(), 10) + ":0"), []string{"library/old.epub", "library/old.epub"}},
	} {
		page, _, err := db.Sources().GetUnembeddedChunks(t.Context(), first.ID, benchModel, at.cursor, 10)
		if err != nil {
			t.Errorf("cursor %q: %v", at.cursor, err)
			continue
		}
		if got := getSourcePaths(page); !slices.Equal(got, at.want) {
			t.Errorf("cursor %q carried on with chunks of %v, want %v", at.cursor, got, at.want)
		}
	}
}

// A cursor this index did not hand out is refused, and nothing is visited.
func TestACursorThisIndexDidNotHandOutIsRefused(t *testing.T) {
	db := openDB(t)
	saveThroughPort(t, db, first, "library/a.epub", thisYear, "bread rises", "yeast ferments")

	for _, cursor := range []port.ChunkCursor{"x", "x:1", "1:x", "x:1:2", "1:x:2", "1:2:x", "1:2:3:4"} {
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

	read, held, err := sources.Reading(ctx, first.ID, "library/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !held {
		t.Fatal("the source saved is not held")
	}
	if read.Size != 4321 || !read.ModTime.Equal(thisYear) || read.Hash != "hash-scan" || read.Producer != "ocr" {
		t.Errorf("the source reads back as %+v", read)
	}
	books, err := sources.Fingerprints(ctx, first.ID, domain.KindBook)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := books["library/scan.pdf"]; !found || len(books) != 1 {
		t.Errorf("the vault's books are %v, want the one saved", books)
	}
}
