package index

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// The embedding walk requests a page of chunks, reads the text of each and
// writes a vector back. The index stores a chunk on a numbered row and the core
// addresses it by a domain.ChunkID, so every row read is converted to an ID and
// every vector written is converted back to a row. These benchmarks measure
// that conversion against the index's native row queries.

// chunksPerPage is the page size the walk requests, as usecase/source does.
const chunksPerPage = 200

func getBenchVault() domain.Vault { return domain.Vault{ID: "01BENCH", Name: "bench", Path: "/bench"} }

func getBenchModel() embedding.Model {
	return embedding.Model{
		Name: "bench-model", Dimensions: 1024, MaxTokens: 512, Pooling: "mean", From: "bench",
	}
}

func openUnembeddedDB(tb testing.TB, sources, per int) *DB {
	tb.Helper()
	ctx := tb.Context()

	path := filepath.Join(tb.TempDir(), "index.db")
	newMigratedTemplate().CopyTo(tb, path)
	db, err := Open(ctx, path)
	if err != nil {
		tb.Fatal(err)
	}
	testsupport.CloseAfter(tb, db)
	if err := db.Vaults().Register(ctx, getBenchVault().ID); err != nil {
		tb.Fatal(err)
	}

	chunks := db.Chunks()
	for s := range sources {
		path := fmt.Sprintf("library/book-%04d.epub", s)
		if err := chunks.SaveSource(ctx, getBenchVault().ID, chunk.Source{
			Path: path, Kind: "book", Size: 1000, MTime: 1, ContentHash: "hash-" + path, Recipe: "epub",
		}); err != nil {
			tb.Fatal(err)
		}
		small := make([]chunk.Chunk, 0, per)
		for i := range per {
			small = append(small, chunk.Chunk{
				Start:  i * 50,
				Length: 50,
				Text:   fmt.Sprintf("passage %d of book %d, some words to index", i, s),
			})
		}
		if err := chunks.ReplaceChunks(ctx, getBenchVault().ID, "book", path, []chunk.Chunk{{
			Start: 0, Length: per * 50, Location: "chapter 1",
			Text:  fmt.Sprintf("book %d whole", s),
			Small: small,
		}}); err != nil {
			tb.Fatal(err)
		}
	}
	return db
}

// BenchmarkUnembedded measures one page of the embedding walk, queried through the
// index's native interface and through the port. Both run the same
// statement over the same rows and differ only in the identifier returned
// for each row.
func BenchmarkUnembedded(b *testing.B) {
	db := openUnembeddedDB(b, 200, 100)
	ctx := b.Context()
	recipe := getBenchModel().Recipe()
	rows := queriesOf(db)
	passages := db.Sources()

	b.Run("Rows", func(b *testing.B) {
		for b.Loop() {
			if _, err := rows.GetUnembeddedChunks(ctx, getBenchVault().ID, recipe, chunk.Cursor{}, chunksPerPage); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Passages", func(b *testing.B) {
		for b.Loop() {
			if _, _, err := passages.GetUnembeddedChunks(ctx, getBenchVault().ID, getBenchModel(), "", chunksPerPage); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSaveVectors writes one page of vectors, in the index's words and
// through the port. The port reads a row number back out of every identifier it
// is given before the write begins.
func BenchmarkSaveVectors(b *testing.B) {
	db := openUnembeddedDB(b, 2, 100)
	ctx := b.Context()

	found, err := db.ChunkVectors().GetUnembeddedChunks(ctx, getBenchVault().ID, getBenchModel().Recipe(), chunk.Cursor{}, chunksPerPage)
	if err != nil {
		b.Fatal(err)
	}
	if len(found) != chunksPerPage {
		b.Fatalf("wanted %d chunks to write vectors for, got %d", chunksPerPage, len(found))
	}

	value := make([]byte, getBenchModel().Dimensions)
	coarse := make([]byte, getBenchModel().Dimensions/8)
	numbered := make([]chunk.Vector, 0, len(found))
	named := make([]embedding.Vector, 0, len(found))
	for i, p := range found {
		hash := []byte(fmt.Sprintf("%032d", i))
		numbered = append(numbered, chunk.Vector{
			Chunk: p.Chunk, Hash: hash, Recipe: getBenchModel().Recipe(), Value: value, Coarse: coarse,
		})
		named = append(named, embedding.Vector{
			ChunkID: chunk.ID(p.Chunk), Hash: hash, Model: getBenchModel(),
			Kind: embedding.QuantizedInt8, Value: value, Coarse: coarse,
		})
	}

	b.Run("Rows", func(b *testing.B) {
		write := db.Chunks()
		for b.Loop() {
			if err := write.SaveVectors(ctx, numbered); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Named", func(b *testing.B) {
		write := db.Sources()
		for b.Loop() {
			if err := write.SaveVectors(ctx, named); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkChunkIdentity is the conversion alone, once per chunk each way.
func BenchmarkChunkIdentity(b *testing.B) {
	b.Run("Name", func(b *testing.B) {
		// b.Loop keeps the calls in its body from being optimized away.
		for i := 0; b.Loop(); i++ {
			chunk.ID(int64(i))
		}
	})
	b.Run("Row", func(b *testing.B) {
		id := chunk.ID(918273645)
		for b.Loop() {
			if _, err := chunk.Row(id); err != nil {
				b.Fatal(err)
			}
		}
	})
}
