package index

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
)

func getQuantizingModel() embedding.Model {
	return embedding.Model{
		Name: "test", Dimensions: 1024, MaxTokens: 256, Pooling: "mean",
		From: "local:library/model.onnx",
	}
}

func atScale(t *testing.T, m embedding.Model, scale float64) string {
	t.Helper()

	recipe, target := m.Recipe(), fmt.Sprintf("@%g", embedding.Int8Scale)
	if !strings.HasSuffix(recipe, target) {
		t.Fatalf("the recipe says nothing about the scale its bytes sit on: %s", recipe)
	}
	return strings.TrimSuffix(recipe, target) + fmt.Sprintf("@%g", scale)
}

// TestAScaleThatMovedBuysTheVectorsAgain is the guarantee the recipe exists
// for. A byte quantized at one scale means something else at another, and the
// index has nothing but the recipe to tell it so.
func TestAScaleThatMovedBuysTheVectorsAgain(t *testing.T) {
	db := openDB(t)
	ctx := t.Context()

	before := atScale(t, getQuantizingModel(), 0.3)
	if before == getQuantizingModel().Recipe() {
		t.Fatalf("two scales, one recipe: %s", before)
	}

	if err := db.Chunks().SaveSource(ctx, getFirstVault().ID, chunk.Source{
		Path: "library/tale.epub", Kind: "book", Size: 1000, MTime: 1,
		ContentHash: "hash-tale", Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().ReplaceChunks(ctx, getFirstVault().ID, "book", "library/tale.epub", []chunk.Chunk{{
		Start: 0, Length: 100, Location: "chapter 1",
		Text: "the tale opens and the tale goes on",
		Small: []chunk.Chunk{
			{Start: 0, Length: 50, Text: "the tale opens"},
			{Start: 50, Length: 50, Text: "the tale goes on"},
		},
	}}); err != nil {
		t.Fatal(err)
	}

	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(ctx, getFirstVault().ID, before, chunk.Cursor{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) != 2 {
		t.Fatalf("%d chunks need a vector before any is made, want 2", len(unembedded))
	}

	hashes := make([][]byte, 0, len(unembedded))
	vectors := make([]chunk.Vector, 0, len(unembedded))
	for _, p := range unembedded {
		hash := hashOf(t, db, p.Chunk)
		hashes = append(hashes, hash)
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: hash, Recipe: before,
			Value: getPreciseBytes(getVectorForSeed(9)), Coarse: bits(9),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatal(err)
	}

	if atBefore, err := db.ChunkVectors().GetUnembeddedChunks(ctx, getFirstVault().ID, before, chunk.Cursor{}, 100); err != nil {
		t.Fatal(err)
	} else if len(atBefore) != 0 {
		t.Fatalf("%d chunks need a vector under the scale they were made at, want none", len(atBefore))
	}

	afterSwitch, err := db.ChunkVectors().GetUnembeddedChunks(ctx, getFirstVault().ID, getQuantizingModel().Recipe(), chunk.Cursor{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterSwitch) != 2 {
		t.Errorf("%d chunks are asked for again after the scale moved, want 2", len(afterSwitch))
	}

	retained, err := db.ChunkVectors().GetKeptVectors(ctx, getQuantizingModel().Recipe(), hashes)
	if err != nil {
		t.Fatal(err)
	}
	for _, hash := range hashes {
		if _, target := retained[hex.EncodeToString(hash)]; target {
			t.Errorf("a vector quantized at another scale is returned for %x", hash)
		}
	}
}
