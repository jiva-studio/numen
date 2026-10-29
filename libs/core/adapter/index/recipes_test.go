package index

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/index/chunk"
)

// A changed recipe leaves its vectors in place, so a user who restores the model
// finds them. The cost is a store carrying both, and this is the only operation
// that removes the old ones.
func TestForgettingTheVectorsOfEveryOtherRecipe(t *testing.T) {
	db := openDB(t)
	ctx := t.Context()

	before := atScale(t, quantising, 0.3)
	now := quantising.Recipe()

	if err := db.Chunks().SaveSource(ctx, first.ID, chunk.Source{
		Path: "library/tale.epub", Kind: "book", Size: 1000, MTime: 1,
		Hash: "hash-tale", Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().ReplaceChunks(ctx, first.ID, "book", "library/tale.epub", []chunk.Chunk{{
		Start: 0, Length: 100, Text: "the tale opens and the tale goes on",
		Small: []chunk.Chunk{
			{Start: 0, Length: 50, Text: "the tale opens"},
			{Start: 50, Length: 50, Text: "the tale goes on"},
		},
	}}); err != nil {
		t.Fatal(err)
	}

	owing, err := db.ChunkQueries().GetUnembeddedChunks(ctx, first.ID, before, 0, 0, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	hashes := make([][]byte, 0, len(owing))
	for _, recipe := range []string{before, now} {
		vectors := make([]chunk.Vector, 0, len(owing))
		for _, p := range owing {
			hash := hashOf(t, db, p.Chunk)
			if recipe == before {
				hashes = append(hashes, hash)
			}
			vectors = append(vectors, chunk.Vector{
				Chunk: p.Chunk, Hash: hash, Recipe: recipe,
				Value: getPreciseBytes(getVectorForSeed(9)), Coarse: bits(9),
			})
		}
		if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := db.DeleteOtherRecipeVectors(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if removed != int64(len(hashes)) {
		t.Errorf("%d vectors went, want %d", removed, len(hashes))
	}

	retained, err := db.ChunkQueries().GetKeptVectors(ctx, now, hashes)
	if err != nil {
		t.Fatal(err)
	}
	for _, hash := range hashes {
		if _, target := retained[hex.EncodeToString(hash)]; !target {
			t.Errorf("the vector under the recipe in use went with the rest: %x", hash)
		}
	}
	limit, err := db.ChunkQueries().GetKeptVectors(ctx, before, hashes)
	if err != nil {
		t.Fatal(err)
	}
	if len(limit) != 0 {
		t.Errorf("%d vectors of the recipe nobody asks for are still here", len(limit))
	}

	// run that has no model to say what it is holding vectors for.
	if removedVal, err := db.DeleteOtherRecipeVectors(ctx, ""); err != nil || removedVal != 0 {
		t.Errorf("forgetting under no recipe took %d vectors out: %v", removedVal, err)
	}
	if err := db.Compact(ctx); err != nil {
		t.Fatal(err)
	}
	if targetVal, err := db.ChunkQueries().GetKeptVectors(ctx, now, hashes); err != nil || len(targetVal) != len(hashes) {
		t.Errorf("%d vectors survived the vacuum, want %d: %v", len(targetVal), len(hashes), err)
	}
}

// A deletion or a compaction that fails is reported.
func TestForgettingAndCompactingReportTheirFailure(t *testing.T) {
	for _, step := range []struct {
		name   string
		breaks string
		run    func(db *DB) error
	}{
		{"forgetting", "DELETE FROM vectors", func(db *DB) error {
			n, err := db.DeleteOtherRecipeVectors(t.Context(), "current")
			if n != 0 {
				t.Errorf("a deletion that failed counted %d", n)
			}
			return err
		}},
		{"compacting", "VACUUM", func(db *DB) error { return db.Compact(t.Context()) }},
	} {
		t.Run(step.name, func(t *testing.T) {
			db := openBreakingDB(t)
			breakOn(t, step.breaks)
			err := step.run(db)
			stopBreaking()
			if !errors.Is(err, errBroken) {
				t.Errorf("answered %v", err)
			}
		})
	}
}
