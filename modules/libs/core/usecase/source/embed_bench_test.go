package source

import (
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func BenchmarkEmbedDuplicates(b *testing.B) {
	cases := []struct {
		name      string
		chunks    int
		dupsRatio float64
	}{
		{name: "chunks=100/dups=0pct", chunks: 100, dupsRatio: 0.0},
		{name: "chunks=100/dups=50pct", chunks: 100, dupsRatio: 0.5},
		{name: "chunks=100/dups=90pct", chunks: 100, dupsRatio: 0.9},
		{name: "chunks=100/dups=100pct", chunks: 100, dupsRatio: 1.0},
		{name: "chunks=500/dups=0pct", chunks: 500, dupsRatio: 0.0},
		{name: "chunks=500/dups=50pct", chunks: 500, dupsRatio: 0.5},
		{name: "chunks=500/dups=90pct", chunks: 500, dupsRatio: 0.9},
		{name: "chunks=500/dups=100pct", chunks: 500, dupsRatio: 1.0},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			uniqueCount := max(1, int(float64(tc.chunks)*(1.0-tc.dupsRatio)))
			uniqueParts := make([]string, uniqueCount)
			for i := range uniqueCount {
				uniqueParts[i] = fmt.Sprintf("Unique section %d %s", i, words(sanskrit, 150))
			}

			parts := make([]string, tc.chunks)
			for i := range tc.chunks {
				parts[i] = uniqueParts[i%uniqueCount]
			}

			shelf := newLibrary()
			shelf.hold(bookPath, domain.KindBook, bookOf(b, "A Book", parts...), 1)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				b.StopTimer()
				index := newStore()
				cutBooks(b, index, shelf, first)
				model := &embedder{dims: dimensions}
				embed := NewEmbed(vaults{first.ID: shelf}, index, index)
				embed.Embedder = model
				embed.BatchCharacters = 4000
				b.StartTimer()

				if _, err := embed.Execute(b.Context(), first); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEmbedBatchDuplicates(b *testing.B) {
	cases := []struct {
		name      string
		chunks    int
		dupsRatio float64
	}{
		{name: "batch=100/dups=0pct", chunks: 100, dupsRatio: 0.0},
		{name: "batch=100/dups=50pct", chunks: 100, dupsRatio: 0.5},
		{name: "batch=100/dups=90pct", chunks: 100, dupsRatio: 0.9},
		{name: "batch=100/dups=100pct", chunks: 100, dupsRatio: 1.0},
		{name: "batch=500/dups=0pct", chunks: 500, dupsRatio: 0.0},
		{name: "batch=500/dups=50pct", chunks: 500, dupsRatio: 0.5},
		{name: "batch=500/dups=90pct", chunks: 500, dupsRatio: 0.9},
		{name: "batch=500/dups=100pct", chunks: 500, dupsRatio: 1.0},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			uniqueCount := max(1, int(float64(tc.chunks)*(1.0-tc.dupsRatio)))
			uniqueTexts := make([]string, uniqueCount)
			for i := range uniqueCount {
				uniqueTexts[i] = fmt.Sprintf("Unique chunk %d %s", i, words(sanskrit, 50))
			}

			texts := make([]string, tc.chunks)
			passages := make([]domain.Passage, tc.chunks)
			for i := range tc.chunks {
				txt := uniqueTexts[i%uniqueCount]
				texts[i] = txt
				passages[i] = domain.Passage{
					ChunkID:   domain.ChunkID(fmt.Sprintf("chunk-%d", i)),
					Source:    "book.epub",
					ChunkHash: hashOf(txt),
				}
			}

			model := port.EmbeddingModel{Name: "fake", Dimensions: dimensions}
			index := newStore()
			embed := NewEmbed(vaults{}, index, index)
			embed.Embedder = &embedder{dims: dimensions}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				prep, err := embed.prepareBatch(b.Context(), model, passages, texts)
				if err != nil {
					b.Fatal(err)
				}
				if _, err := embed.inferBatch(b.Context(), prep); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
