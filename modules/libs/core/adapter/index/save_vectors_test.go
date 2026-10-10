package index

import (
	"encoding/hex"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
)

func TestSaveVectorsBatchedCommit(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)

	vault := first.ID
	path := "doc.md"

	if err := db.Chunks().SaveSource(ctx, vault, chunk.Source{
		Path: path, Kind: "note", Size: 100, MTime: 1, Hash: "h" + path, Recipe: "text",
	}); err != nil {
		t.Fatal(err)
	}

	// Create 1500 small chunks to test batch sizes of 1000
	const numChunks = 1500
	chunks := make([]chunk.Chunk, 1)
	chunks[0] = chunk.Chunk{Start: 0, Length: numChunks * 10, Text: "whole", Small: make([]chunk.Chunk, numChunks)}
	for i := range numChunks {
		chunks[0].Small[i] = chunk.Chunk{Start: i * 10, Length: 10, Text: "chunk" + hex.EncodeToString([]byte{byte(i % 256), byte(i / 256)})}
	}

	if err := db.Chunks().ReplaceChunks(ctx, vault, "note", path, chunks); err != nil {
		t.Fatal(err)
	}

	owing, err := db.ChunkQueries().GetUnembeddedChunks(ctx, vault, "model", 0, 0, 0, numChunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) != numChunks {
		t.Fatalf("expected %d chunks needing embedding, got %d", numChunks, len(owing))
	}

	vectors := make([]chunk.Vector, len(owing))
	for i, o := range owing {
		hash, _ := hex.DecodeString(o.ChunkHash)
		vectors[i] = chunk.Vector{
			Chunk:  o.Chunk,
			Hash:   hash,
			Recipe: "model",
			Value:  make([]byte, 1024),
			Coarse: make([]byte, 1024/8),
		}
	}

	// Ensure our batch size is applied
	originalBatchSize := chunk.VectorBatchSize
	chunk.VectorBatchSize = 1000
	defer func() { chunk.VectorBatchSize = originalBatchSize }()

	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatalf("batched insertion failed: %v", err)
	}

	// Verify all were written
	owingAfter, err := db.ChunkQueries().GetUnembeddedChunks(ctx, vault, "model", 0, 0, 0, numChunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(owingAfter) != 0 {
		t.Errorf("expected 0 unembedded chunks after save, got %d", len(owingAfter))
	}
}
