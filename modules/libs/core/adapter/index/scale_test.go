package index

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var quantising = port.EmbeddingModel{
	Name: "test", Dimensions: 1024, MaxTokens: 256, Pooling: "mean",
	From: "local:library/model.onnx",
}

// atScale is the recipe this model had while its bytes sat on another grid. It
// reads the scale out of the recipe in force, so a recipe that stopped carrying
// the scale is a failure here and not a test that quietly passes.
func atScale(t *testing.T, m port.EmbeddingModel, scale float64) string {
	t.Helper()

	recipe, held := m.Recipe(), fmt.Sprintf("@%g", port.Int8Scale)
	if !strings.HasSuffix(recipe, held) {
		t.Fatalf("the recipe says nothing about the scale its bytes sit on: %s", recipe)
	}
	return strings.TrimSuffix(recipe, held) + fmt.Sprintf("@%g", scale)
}

// TestAScaleThatMovedBuysTheVectorsAgain is the guarantee the recipe exists
// for. A byte quantised at one scale means something else at another, and the
// index has nothing but the recipe to tell it so.
func TestAScaleThatMovedBuysTheVectorsAgain(t *testing.T) {
	db := openDB(t)
	ctx := t.Context()

	before := atScale(t, quantising, 0.3)
	if before == quantising.Recipe() {
		t.Fatalf("two scales, one recipe: %s", before)
	}

	if err := db.Chunks().SaveSource(ctx, first.ID, chunk.Source{
		Path: "library/tale.epub", Kind: "book", Size: 1000, MTime: 1,
		Hash: "hash-tale", Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().ReplaceChunks(ctx, first.ID, "book", "library/tale.epub", []chunk.Chunk{{
		Start: 0, Length: 100, Location: "chapter 1",
		Text: "the tale opens and the tale goes on",
		Small: []chunk.Chunk{
			{Start: 0, Length: 50, Text: "the tale opens"},
			{Start: 50, Length: 50, Text: "the tale goes on"},
		},
	}}); err != nil {
		t.Fatal(err)
	}

	owing, err := db.ChunkQueries().Unembedded(ctx, first.ID, before, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) != 2 {
		t.Fatalf("%d chunks owe a vector before any is made, want 2", len(owing))
	}

	hashes := make([][]byte, 0, len(owing))
	vectors := make([]chunk.Vector, 0, len(owing))
	for _, p := range owing {
		hash := hashOf(t, db, p.Chunk)
		hashes = append(hashes, hash)
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: hash, Recipe: before,
			Value: precise(direction(9)), Coarse: bits(9),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatal(err)
	}

	if owed, err := db.ChunkQueries().Unembedded(ctx, first.ID, before, 0, 100); err != nil {
		t.Fatal(err)
	} else if len(owed) != 0 {
		t.Fatalf("%d chunks owe a vector under the scale they were made at, want none", len(owed))
	}

	owed, err := db.ChunkQueries().Unembedded(ctx, first.ID, quantising.Recipe(), 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(owed) != 2 {
		t.Errorf("%d chunks are asked for again after the scale moved, want 2", len(owed))
	}

	kept, err := db.ChunkQueries().Kept(ctx, quantising.Recipe(), hashes)
	if err != nil {
		t.Fatal(err)
	}
	for _, hash := range hashes {
		if _, held := kept[hex.EncodeToString(hash)]; held {
			t.Errorf("a vector quantised at another scale is handed back for %x", hash)
		}
	}
}
