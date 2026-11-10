package chunk_test

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const dimensions = 1024

func preciseBytes(v []float32) []byte {
	q := vector.Bytes(vector.Normalize(v))
	out := make([]byte, len(q))
	for i, x := range q {
		out[i] = byte(x)
	}
	return out
}

func seedCoarseRanks(t *testing.T, exact int) (domain.VaultID, *chunk.Search, []float32, int) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ctx := t.Context()

	db, err := index.Open(ctx, filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	const vault domain.VaultID = "01RANKS"
	if err := db.Vaults().Register(ctx, vault); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().SaveSource(ctx, vault, chunk.Source{
		Path: "library/ranks.epub", Kind: "book", Size: 1000, MTime: 1, ContentHash: "hash-ranks", Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	var small []chunk.Chunk
	for rank := 1; rank <= 10; rank++ {
		small = append(small, chunk.Chunk{Start: rank * 10, Length: 10, Text: fmt.Sprintf("passage number %d", rank)})
	}
	if err := db.Chunks().ReplaceChunks(ctx, vault, "book", "library/ranks.epub", []chunk.Chunk{{
		Start: 0, Length: 200, Location: "chapter 1", Text: "the whole of it", Small: small,
	}}); err != nil {
		t.Fatal(err)
	}

	query := make([]float32, dimensions)
	unrelated := make([]float32, dimensions)
	for i := range query {
		query[i] = -1
		unrelated[i] = -1
		if i < dimensions/2 {
			unrelated[i] = 1
		}
	}

	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(ctx, vault, "model", chunk.Cursor{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	var vectors []chunk.Vector
	for _, p := range unembedded {
		if p.Length != 10 {
			continue
		}
		rank := p.Start / 10
		hash, err := hex.DecodeString(p.ChunkHash)
		if err != nil {
			t.Fatal(err)
		}
		coarse := make([]byte, dimensions/8)
		for b := range rank {
			coarse[b] = 1
		}
		value := preciseBytes(unrelated)
		if rank == exact {
			value = preciseBytes(query)
		}
		vectors = append(vectors, chunk.Vector{Chunk: p.Chunk, Hash: hash, Recipe: "model", Value: value, Coarse: coarse})
	}
	if len(vectors) != 10 {
		t.Fatalf("seeded %d small chunks, want 10", len(vectors))
	}
	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatal(err)
	}
	return vault, db.ChunkSearch(), query, exact * 10
}

func TestTheCoarsePassKeepsEightCandidatesForEachAnswer(t *testing.T) {
	vault, queries, query, hitAt := seedCoarseRanks(t, 8)
	got, err := queries.FindNearest(t.Context(), vault, sourcedomain.NearestQuery{Recipe: "model", Vector: query, Limit: 1, Floor: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].HitAt != hitAt {
		t.Fatalf("got %+v, want the passage at %d", got, hitAt)
	}
}

func TestTheCoarsePassDropsTheNinthCandidateOfOneAnswer(t *testing.T) {
	vault, queries, query, _ := seedCoarseRanks(t, 9)
	got, err := queries.FindNearest(t.Context(), vault, sourcedomain.NearestQuery{Recipe: "model", Vector: query, Limit: 1, Floor: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %+v, want nothing: the nearest by full precision stands ninth by bits", got)
	}
}
