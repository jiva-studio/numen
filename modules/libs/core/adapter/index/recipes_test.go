package index

import (
	"encoding/hex"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
)

// A recipe that moved leaves its vectors where they are, so a person who sets
// the model back finds them. What they cost is a store carrying both, and this
// is the one thing that takes the old ones out.
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

	owing, err := db.ChunkQueries().GetUnembeddedChunks(ctx, first.ID, before, 0, 100)
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
				Value: precise(direction(9)), Coarse: bits(9),
			})
		}
		if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
			t.Fatal(err)
		}
	}

	gone, err := db.ForgetOtherRecipes(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if gone != int64(len(hashes)) {
		t.Errorf("%d vectors went, want %d", gone, len(hashes))
	}

	kept, err := db.ChunkQueries().GetKeptVectors(ctx, now, hashes)
	if err != nil {
		t.Fatal(err)
	}
	for _, hash := range hashes {
		if _, held := kept[hex.EncodeToString(hash)]; !held {
			t.Errorf("the vector under the recipe in use went with the rest: %x", hash)
		}
	}
	under, err := db.ChunkQueries().GetKeptVectors(ctx, before, hashes)
	if err != nil {
		t.Fatal(err)
	}
	if len(under) != 0 {
		t.Errorf("%d vectors of the recipe nobody asks for are still here", len(under))
	}

	// A recipe nobody named takes nothing out: the store is not emptied by a
	// run that has no model to say what it is holding vectors for.
	if gone, err := db.ForgetOtherRecipes(ctx, ""); err != nil || gone != 0 {
		t.Errorf("forgetting under no recipe took %d vectors out: %v", gone, err)
	}
	if err := db.Compact(ctx); err != nil {
		t.Fatal(err)
	}
	if held, err := db.ChunkQueries().GetKeptVectors(ctx, now, hashes); err != nil || len(held) != len(hashes) {
		t.Errorf("%d vectors survived the vacuum, want %d: %v", len(held), len(hashes), err)
	}
}
