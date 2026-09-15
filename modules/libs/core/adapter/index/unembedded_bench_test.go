package index

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The embedding walk asks for a page of chunks, reads the text of each and
// writes a vector back. The index holds a chunk on a numbered row and the core
// addresses it by a domain.ChunkID, so every row read is spelled out and every
// vector written is read back in. What that costs is measured here against the
// query and the write it sits inside.

// chunksPerPage is what the walk asks for at a time, as usecase/source does.
const chunksPerPage = 200

// benchVault holds every chunk these benchmarks read.
var benchVault = domain.Vault{ID: "01BENCH", Name: "bench", Path: "/bench"}

// benchModel is a model nothing was embedded with, so every chunk owes it one.
var benchModel = port.EmbeddingModel{
	Name: "bench-model", Dimensions: 1024, MaxTokens: 512, Pooling: "mean", From: "bench",
}

// openUnembeddedDB is an index holding one vault of `sources` sources, each cut
// into a large chunk with `per` small ones inside it, and no vectors at all.
func openUnembeddedDB(b *testing.B, sources, per int) *DB {
	b.Helper()
	ctx := b.Context()

	path := filepath.Join(b.TempDir(), "index.db")
	migrated.CopyTo(b, path)
	db, err := Open(ctx, path)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { db.Close() })
	if err := db.Vaults().Register(ctx, benchVault.ID); err != nil {
		b.Fatal(err)
	}

	chunks := db.Chunks()
	for s := range sources {
		path := fmt.Sprintf("library/book-%04d.epub", s)
		if err := chunks.SaveSource(ctx, benchVault.ID, chunk.Source{
			Path: path, Kind: "book", Size: 1000, MTime: 1, Hash: "hash-" + path, Recipe: "epub",
		}); err != nil {
			b.Fatal(err)
		}
		small := make([]chunk.Chunk, 0, per)
		for i := range per {
			small = append(small, chunk.Chunk{
				Start:  i * 50,
				Length: 50,
				Text:   fmt.Sprintf("passage %d of book %d, some words to index", i, s),
			})
		}
		if err := chunks.ReplaceChunks(ctx, benchVault.ID, "book", path, []chunk.Chunk{{
			Start: 0, Length: per * 50, Location: "chapter 1",
			Text:  fmt.Sprintf("book %d whole", s),
			Small: small,
		}}); err != nil {
			b.Fatal(err)
		}
	}
	return db
}

// BenchmarkUnembedded is one page of the embedding walk, asked of the index in
// its own words and asked of it through the port. The two run the same
// statement over the same rows and differ by the identifier each row is handed
// back under.
func BenchmarkUnembedded(b *testing.B) {
	db := openUnembeddedDB(b, 200, 100)
	ctx := b.Context()
	recipe := benchModel.Recipe()
	rows := db.ChunkQueries()
	passages := db.Sources()

	b.Run("Rows", func(b *testing.B) {
		for b.Loop() {
			if _, err := rows.GetUnembeddedChunks(ctx, benchVault.ID, recipe, 0, chunksPerPage); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Passages", func(b *testing.B) {
		for b.Loop() {
			if _, _, err := passages.GetUnembeddedChunks(ctx, benchVault.ID, benchModel, "", chunksPerPage); err != nil {
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

	found, err := db.ChunkQueries().GetUnembeddedChunks(ctx, benchVault.ID, benchModel.Recipe(), 0, chunksPerPage)
	if err != nil {
		b.Fatal(err)
	}
	if len(found) != chunksPerPage {
		b.Fatalf("wanted %d chunks to write vectors for, got %d", chunksPerPage, len(found))
	}

	value := make([]byte, benchModel.Dimensions)
	coarse := make([]byte, benchModel.Dimensions/8)
	numbered := make([]chunk.Vector, 0, len(found))
	named := make([]port.Vector, 0, len(found))
	for i, p := range found {
		hash := []byte(fmt.Sprintf("%032d", i))
		numbered = append(numbered, chunk.Vector{
			Chunk: p.Chunk, Hash: hash, Recipe: benchModel.Recipe(), Value: value, Coarse: coarse,
		})
		named = append(named, port.Vector{
			ChunkID: chunk.ID(p.Chunk), Hash: hash, Model: benchModel,
			Kind: port.QuantisedInt8, Value: value, Coarse: coarse,
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
		var id domain.ChunkID
		for i := 0; b.Loop(); i++ {
			id = chunk.ID(int64(i))
		}
		sink = string(id)
	})
	b.Run("Row", func(b *testing.B) {
		id := chunk.ID(918273645)
		var row int64
		for b.Loop() {
			r, err := chunk.Row(id)
			if err != nil {
				b.Fatal(err)
			}
			row = r
		}
		sink = fmt.Sprint(row)
	})
}

// sink keeps the conversions above from being optimised away.
var sink string
