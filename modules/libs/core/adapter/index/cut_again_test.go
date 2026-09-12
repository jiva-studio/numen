package index

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
)

// said is the words a generated cutting draws its chunks from. It is small, so
// one text stands twice in a source about as often as it stands once, and a
// large chunk holds what a chunk inside one holds.
var said = []string{"compost", "leaves", "peelings", "turned", "rot down", ""}

// drawChunks generates a source cut into large chunks with small chunks inside
// them. What a chunk holds is what identifies it, so the words are what varies;
// where it stands is what a cut moves, and it moves with the text.
func drawChunks(t *rapid.T, name string) []chunk.Chunk {
	word := rapid.SampledFrom(said)
	inside := rapid.Custom(func(t *rapid.T) chunk.Chunk {
		return chunk.Chunk{Text: word.Draw(t, "small")}
	})
	large := rapid.Custom(func(t *rapid.T) chunk.Chunk {
		return chunk.Chunk{
			Text:  word.Draw(t, "large"),
			Small: rapid.SliceOfN(inside, 0, 3).Draw(t, "inside"),
		}
	})
	out := rapid.SliceOfN(large, 0, 4).Draw(t, name)

	at := 0
	for i := range out {
		out[i].Start, out[i].Length = at, len(out[i].Text)
		held := at
		for j := range out[i].Small {
			out[i].Small[j].Start = held
			out[i].Small[j].Length = len(out[i].Small[j].Text)
			held += len(out[i].Small[j].Text)
		}
		at += len(out[i].Text)
	}
	return out
}

// textOf is what a chunk has to hold to be held on a row: the same text, cut at
// the same size.
type textOf struct {
	hash  string
	small bool
}

// countChunks is what a cut asks the index to hold, as many times as it asks.
func countChunks(chunks []chunk.Chunk) map[textOf]int {
	out := map[textOf]int{}
	for _, large := range chunks {
		out[textOf{hash: hashText(large.Text)}]++
		for _, small := range large.Small {
			out[textOf{hash: hashText(small.Text), small: true}]++
		}
	}
	return out
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// getSourceRows is the rows one source stands on, by the text and the size each
// holds.
func getSourceRows(t *testing.T, db *DB, source int64) map[textOf][]int64 {
	t.Helper()
	cursor, err := db.read.QueryContext(t.Context(),
		`SELECT id, hash, parent_id IS NOT NULL FROM chunks WHERE source_id = ? ORDER BY id`, source)
	if err != nil {
		t.Fatal(err)
	}
	defer cursor.Close()

	out := map[textOf][]int64{}
	for cursor.Next() {
		var row int64
		var key textOf
		if err := cursor.Scan(&row, &key.hash, &key.small); err != nil {
			t.Fatal(err)
		}
		out[key] = append(out[key], row)
	}
	if err := cursor.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// howMany is what a set of rows comes to, one key at a time.
func howMany(rows map[textOf][]int64) map[textOf]int {
	out := make(map[textOf]int, len(rows))
	for key, held := range rows {
		out[key] = len(held)
	}
	return out
}

// cutSource puts one book in and cuts it into the chunks given, answering with
// the row the source stands on.
func cutSource(t *testing.T, db *DB, path string, chunks []chunk.Chunk) int64 {
	t.Helper()
	if err := db.Chunks().SaveSource(t.Context(), first.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, Hash: "hash-" + path, Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().ReplaceChunks(t.Context(), first.ID, "book", path, chunks); err != nil {
		t.Fatal(err)
	}
	var source int64
	if err := db.read.QueryRowContext(t.Context(),
		`SELECT id FROM sources WHERE path = ?`, path).Scan(&source); err != nil {
		t.Fatal(err)
	}
	return source
}

// A chunk is identified by the hash of its text, and cutting a source again is
// a comparison against that column: a chunk whose hash is on a row of this
// source keeps that row, a chunk whose hash is on no row is a new chunk, and a
// row whose hash is in no chunk is a chunk that is gone.
//
// A row is claimed once, so text occurring twice in one source is two rows and
// stays two, and the same text cut at two sizes is a row at each. A kept row
// keeps the vector bought for it.
func TestCuttingASourceAgainKeepsTheRowsOfTheTextItStillHolds(t *testing.T) {
	t.Parallel()
	outer := t

	var kept, twice, gone int
	rapid.Check(t, func(t *rapid.T) {
		was := drawChunks(t, "the first cut")
		now := drawChunks(t, "the second cut")

		db := openDB(outer)
		source := cutSource(outer, db, "library/a.epub", was)
		before := getSourceRows(outer, db, source)

		if err := db.Chunks().ReplaceChunks(outer.Context(), first.ID, "book", "library/a.epub", now); err != nil {
			t.Fatal(err)
		}
		after := getSourceRows(outer, db, source)

		// The source stands on exactly the chunks this cut asked for.
		if want := countChunks(now); !maps.Equal(howMany(after), want) {
			t.Fatalf("%v cut again as %v stands on %v, want %v",
				was, now, howMany(after), want)
		}

		// The text it held before, at the size it held it at, keeps the rows it
		// stood on — as many of them as both cuts ask for.
		for key, held := range after {
			survived := 0
			for _, row := range held {
				for _, already := range before[key] {
					if row == already {
						survived++
						break
					}
				}
			}
			if want := min(len(before[key]), len(held)); survived != want {
				t.Fatalf("%d of the rows holding %v survived %v cut again as %v, want %d",
					survived, key, was, now, want)
			}
			kept += survived
			if len(held) > 1 {
				twice++
			}
		}
		for key, held := range before {
			if len(held) > len(after[key]) {
				gone++
			}
		}
	})
	// A run in which no row was ever kept, or one where a source never held one
	// text twice, or one where nothing was ever cut away, asks nothing of any of
	// the three rules.
	if kept < 40 || twice < 20 || gone < 50 {
		t.Fatalf("%d rows were kept, %d texts stood twice in one source, "+
			"and %d texts were cut away", kept, twice, gone)
	}
}
