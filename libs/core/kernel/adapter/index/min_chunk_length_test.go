package index

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
)

func getShortPieceLengths() []int { return []int{5, 39, 40, 41} }

func cutPieces(t *testing.T, db *DB) {
	t.Helper()
	ctx := t.Context()

	if err := db.Chunks().SaveSource(ctx, getFirstVault().ID, chunk.Source{
		Path: "library/pages.epub", Kind: "book", Size: 1000, MTime: 1,
		ContentHash: "hash-pages", Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	small := make([]chunk.Chunk, 0, len(getShortPieceLengths()))
	start := 0
	for i, length := range getShortPieceLengths() {
		small = append(small, chunk.Chunk{Start: start, Length: length, Text: strings.Repeat(string(rune('a'+i)), length)})
		start += length
	}
	if err := db.Chunks().ReplaceChunks(ctx, getFirstVault().ID, "book", "library/pages.epub", []chunk.Chunk{{
		Start: 0, Length: start, Text: strings.Repeat("x", start), Small: small,
	}}); err != nil {
		t.Fatal(err)
	}
}

func embedPieces(t *testing.T, db *DB, recipe string) {
	t.Helper()
	cutPieces(t, db)
	ctx := t.Context()
	pending, err := db.ChunkVectors().GetUnembeddedChunks(ctx, getFirstVault().ID, recipe, chunk.Cursor{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != len(getShortPieceLengths()) {
		t.Fatalf("%d small chunks to embed, want %d", len(pending), len(getShortPieceLengths()))
	}
	vectors := make([]chunk.Vector, 0, len(pending))
	for _, p := range pending {
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: hashOf(t, db, p.Chunk), Recipe: recipe,
			Value: getPreciseBytes(getVectorForSeed(9)), Coarse: bits(9),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatal(err)
	}
}

func pendingLengths(t *testing.T, db *DB, recipe string) []int {
	t.Helper()
	pending, err := db.ChunkVectors().GetUnembeddedChunks(t.Context(), getFirstVault().ID, recipe, chunk.Cursor{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	var out []int
	for _, p := range pending {
		out = append(out, p.Length)
	}
	slices.Sort(out)
	return out
}

func keptLengths(t *testing.T, db *DB, recipe string) []int {
	t.Helper()
	var out []int
	for i, length := range getShortPieceLengths() {
		hash := hashOf(t, db, firstChunkOfLength(t, db, length))
		kept, err := db.ChunkVectors().GetKeptVectors(t.Context(), recipe, [][]byte{hash})
		if err != nil {
			t.Fatal(err)
		}
		if len(kept) > 0 {
			out = append(out, getShortPieceLengths()[i])
		}
	}
	return out
}

func firstChunkOfLength(t *testing.T, db *DB, length int) int64 {
	t.Helper()
	var id int64
	if err := db.write.QueryRowContext(t.Context(),
		`SELECT id FROM chunks WHERE parent_id IS NOT NULL AND length = ?`, length).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// A chunk shorter than the minimum loses its vector and stops being counted, a chunk at the minimum
// or longer keeps both, and the large chunk is never touched. Zero keeps everything.
func TestApplyMinChunkLength(t *testing.T) {
	cases := []struct {
		name        string
		minimum     int
		wantTarget  int64
		wantKept    []int
		wantRemoved int64
	}{
		{name: "zero embeds everything", minimum: 0, wantTarget: 4, wantKept: []int{5, 39, 40, 41}, wantRemoved: 0},
		{name: "one keeps every chunk", minimum: 1, wantTarget: 4, wantKept: []int{5, 39, 40, 41}, wantRemoved: 0},
		{name: "below the shortest is the same as zero", minimum: 5, wantTarget: 4, wantKept: []int{5, 39, 40, 41}, wantRemoved: 0},
		{name: "a chunk of the minimum length is kept", minimum: 40, wantTarget: 2, wantKept: []int{40, 41}, wantRemoved: 2},
		{name: "one above a length removes it", minimum: 41, wantTarget: 1, wantKept: []int{41}, wantRemoved: 3},
		{name: "above them all removes all", minimum: 1000, wantTarget: 0, wantKept: nil, wantRemoved: 4},
		{name: "a negative minimum is zero", minimum: -3, wantTarget: 4, wantKept: []int{5, 39, 40, 41}, wantRemoved: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := openDB(t)
			recipe := getQuantizingModel().Recipe()
			embedPieces(t, db, recipe)

			removed, err := db.ApplyMinChunkLength(t.Context(), tc.minimum)
			if err != nil {
				t.Fatal(err)
			}
			if removed != tc.wantRemoved {
				t.Errorf("removed %d coarse rows, want %d", removed, tc.wantRemoved)
			}

			target, embedded, err := db.ChunkVectors().GetProgress(t.Context(), getFirstVault().ID, recipe)
			if err != nil {
				t.Fatal(err)
			}
			if target != tc.wantTarget || embedded != tc.wantTarget {
				t.Errorf("progress = %d of %d, want %d of %d", embedded, target, tc.wantTarget, tc.wantTarget)
			}
			if got := keptLengths(t, db, recipe); !slices.Equal(got, tc.wantKept) {
				t.Errorf("vectors kept for lengths %v, want %v", got, tc.wantKept)
			}
			if got := pendingLengths(t, db, recipe); len(got) != 0 {
				t.Errorf("chunks of lengths %v are chosen to embed, want none", got)
			}
			if rows := countRows(t, db, `SELECT count(*) FROM chunks WHERE parent_id IS NULL`); rows != 1 {
				t.Errorf("%d large chunks, want 1", rows)
			}
			if rows := countRows(t, db, `SELECT count(*) FROM chunks_vec`); int64(rows) != tc.wantTarget {
				t.Errorf("%d coarse rows, want %d", rows, tc.wantTarget)
			}
		})
	}
}

// The chunks that are chosen to embed are the ones at or above the minimum, and lowering the minimum
// chooses the ones that were left out again, without the vectors of the others being asked for twice.
func TestMinChunkLengthDecidesWhichChunksAreChosenToEmbed(t *testing.T) {
	cases := []struct {
		minimum int
		want    []int
	}{
		{minimum: 0, want: []int{5, 39, 40, 41}},
		{minimum: 40, want: []int{40, 41}},
		{minimum: 41, want: []int{41}},
		{minimum: 42, want: nil},
	}
	for _, tc := range cases {
		db := openDB(t)
		recipe := getQuantizingModel().Recipe()
		cutPieces(t, db)
		if _, err := db.ApplyMinChunkLength(t.Context(), tc.minimum); err != nil {
			t.Fatal(err)
		}
		if got := pendingLengths(t, db, recipe); !slices.Equal(got, tc.want) {
			t.Errorf("minimum %d: chosen lengths %v, want %v", tc.minimum, got, tc.want)
		}
	}
}

// Setting the same minimum again removes nothing, and lowering it leaves the vectors that are
// there; the chunks that lost theirs are chosen to embed again.
func TestLoweringTheMinimumChoosesTheChunksThatLostTheirVectors(t *testing.T) {
	db := openDB(t)
	recipe := getQuantizingModel().Recipe()
	embedPieces(t, db, recipe)

	if _, err := db.ApplyMinChunkLength(t.Context(), 40); err != nil {
		t.Fatal(err)
	}
	if removed, err := db.ApplyMinChunkLength(t.Context(), 40); err != nil || removed != 0 {
		t.Fatalf("the same minimum removed %d, %v; want nothing", removed, err)
	}
	if _, err := db.ApplyMinChunkLength(t.Context(), 0); err != nil {
		t.Fatal(err)
	}
	if got := pendingLengths(t, db, recipe); !slices.Equal(got, []int{5, 39}) {
		t.Errorf("chosen lengths %v, want [5 39]", got)
	}
	target, embedded, err := db.ChunkVectors().GetProgress(t.Context(), getFirstVault().ID, recipe)
	if err != nil {
		t.Fatal(err)
	}
	if target != 4 || embedded != 2 {
		t.Errorf("progress = %d of %d, want 2 of 4", embedded, target)
	}
}
