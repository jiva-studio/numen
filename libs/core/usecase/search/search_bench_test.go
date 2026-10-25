package search_test

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

type benchEmbedder struct {
	model embeddingport.EmbeddingModel
}

func newBenchEmbedder(model embeddingport.EmbeddingModel) *benchEmbedder {
	return &benchEmbedder{model: model}
}

func (e *benchEmbedder) GetModel() embeddingport.EmbeddingModel {
	return e.model
}

func (e *benchEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = generateDeterministicVector(t, e.model.Dimensions)
	}
	return out, nil
}

func (e *benchEmbedder) Close() error {
	return nil
}

func generateDeterministicVector(text string, dims int) []float32 {
	v := make([]float32, dims)
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return v
	}
	for _, w := range words {
		h := fnv.New64a()
		_, _ = h.Write([]byte(w))
		seed := h.Sum64()
		r := rand.New(rand.NewPCG(seed, seed^0x5555555555555555))
		for range 16 {
			idx := r.IntN(dims)
			if r.IntN(2) == 0 {
				v[idx] += 1.0
			} else {
				v[idx] -= 1.0
			}
		}
	}
	var norm float64
	for _, val := range v {
		norm += float64(val * val)
	}
	if norm == 0 {
		return v
	}
	invNorm := float32(1.0 / math.Sqrt(norm))
	for i := range v {
		v[i] *= invNorm
	}
	return v
}

type benchCorpus struct {
	vault      domain.Vault
	db         *container.Index
	searchMock search.Search
	mockModel  embeddingport.Embedder
	passages   sourceport.PassageQueries
	candidates []domain.Passage
	recipe     string
	queryVec   []float32
}

func newBenchCorpus(b *testing.B, notes int) *benchCorpus {
	b.Helper()
	ctx := b.Context()

	v := testsupport.GenerateVault(b, notes)
	embedCfg := container.DefaultEmbedding()
	model := embeddingport.EmbeddingModel{
		Name:       embedCfg.Model.Name,
		Dimensions: embedCfg.Model.Dimensions,
		MaxTokens:  embedCfg.Model.MaxTokens,
		Pooling:    embedCfg.Model.Pooling,
	}

	dbPath := filepath.Join(b.TempDir(), "search_bench.db")
	cfg := container.Config{
		IndexPath: dbPath,
		Embedding: embedCfg,
	}
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = db.Close() })

	if err := db.Vaults().Register(ctx, v.ID); err != nil {
		b.Fatal(err)
	}
	if err := db.FitVectors(ctx, model.Dimensions, model.Recipe()); err != nil {
		b.Fatal(err)
	}

	scan := cfg.Scan(db)
	if _, err := scan.Execute(ctx, v); err != nil {
		b.Fatal(err)
	}

	mock := newBenchEmbedder(model)
	embedderUseCase, err := cfg.Embed(db, mock, v)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := embedderUseCase.Execute(ctx, v); err != nil {
		b.Fatal(err)
	}

	searchMock := cfg.NewSearch(db, mock, nil)

	passages := db.Passages()
	candidates, err := passages.FindLexical(ctx, v.ID, "entropy thermodynamics", nil, 20, false)
	if err != nil {
		b.Fatal(err)
	}

	queryVectors, err := mock.Embed(ctx, []string{"entropy thermodynamics"})
	if err != nil {
		b.Fatal(err)
	}

	return &benchCorpus{
		vault:      v,
		db:         db,
		searchMock: searchMock,
		mockModel:  mock,
		passages:   passages,
		candidates: candidates,
		recipe:     model.Recipe(),
		queryVec:   queryVectors[0],
	}
}

func BenchmarkSearch(b *testing.B) {
	const (
		queryLexical = "entropy thermodynamics"
		queryTyping  = "thermodynam"
	)

	for _, notes := range []int{1_000, 10_000} {
		b.Run(fmt.Sprintf("notes=%d", notes), func(b *testing.B) {
			c := newBenchCorpus(b, notes)
			ctx := b.Context()

			b.Run("Hybrid", func(b *testing.B) {
				p := search.Parameters{Floor: 0.1}
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					res, err := c.searchMock.Execute(ctx, c.vault, queryLexical, p)
					if err != nil {
						b.Fatal(err)
					}
					if len(res) == 0 {
						b.Fatal("hybrid search returned no passages")
					}
				}
			})

			b.Run("Passages", func(b *testing.B) {
				p := search.Parameters{Limit: 20, Each: 5, Floor: 0.1}
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					res, err := c.searchMock.Execute(ctx, c.vault, queryLexical, p)
					if err != nil {
						b.Fatal(err)
					}
					if len(res) == 0 {
						b.Fatal("passages search returned no passages")
					}
				}
			})

			b.Run("Lexical", func(b *testing.B) {
				p := search.Parameters{Limit: 20, Lexical: 100, Dense: 0, Named: 0}
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					res, err := c.searchMock.Execute(ctx, c.vault, queryLexical, p)
					if err != nil {
						b.Fatal(err)
					}
					if len(res) == 0 {
						b.Fatal("lexical search returned no passages")
					}
				}
			})

			b.Run("Dense/Mock", func(b *testing.B) {
				p := search.Parameters{Limit: 20, Lexical: 0, Dense: 20, Named: 0, Floor: 0.1}
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					res, err := c.searchMock.Execute(ctx, c.vault, queryLexical, p)
					if err != nil {
						b.Fatal(err)
					}
					if len(res) == 0 {
						b.Fatal("dense mock search returned no passages")
					}
				}
			})

			b.Run("Typing/Hybrid", func(b *testing.B) {
				p := search.GetTypingParameters(search.Hybrid, 20)
				p.Floor = 0.1
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					res, err := c.searchMock.Execute(ctx, c.vault, queryTyping, p)
					if err != nil {
						b.Fatal(err)
					}
					if len(res) == 0 {
						b.Fatal("typing hybrid search returned no passages")
					}
				}
			})

			b.Run("Typing/Lexical", func(b *testing.B) {
				p := search.GetTypingParameters(search.Lexical, 20)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					res, err := c.searchMock.Execute(ctx, c.vault, queryTyping, p)
					if err != nil {
						b.Fatal(err)
					}
					if len(res) == 0 {
						b.Fatal("typing lexical search returned no passages")
					}
				}
			})
		})
	}
}

func BenchmarkComponents(b *testing.B) {
	const queryLexical = "entropy thermodynamics"

	for _, notes := range []int{1_000, 10_000} {
		b.Run(fmt.Sprintf("notes=%d", notes), func(b *testing.B) {
			c := newBenchCorpus(b, notes)
			ctx := b.Context()

			b.Run("PassagesLexical", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					res, err := c.passages.FindLexical(ctx, c.vault.ID, queryLexical, nil, 100, false)
					if err != nil {
						b.Fatal(err)
					}
					if len(res) == 0 {
						b.Fatal("lexical query returned no passages")
					}
				}
			})

			b.Run("PassagesGetNamedPassages", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					if _, err := c.passages.GetNamedPassages(ctx, c.vault.ID, "Section", nil, 20, false); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("FindNearest/VectorScanAndRerank", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					res, err := c.passages.FindNearest(ctx, c.vault.ID, c.recipe, c.queryVec, nil, 20, 0.1)
					if err != nil {
						b.Fatal(err)
					}
					if len(res) == 0 {
						b.Fatal("findNearest returned no passages")
					}
				}
			})

			b.Run("QueryEmbedding/Mock", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					_, err := c.mockModel.Embed(ctx, []string{queryLexical})
					if err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("QueryEmbedding/Cached", func(b *testing.B) {
				cache := search.NewQueryCache(256)
				cache.Put(c.recipe, queryLexical, c.queryVec)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					vec, ok := cache.Get(c.recipe, queryLexical)
					if !ok || len(vec) == 0 {
						b.Fatal("cached query vector not found")
					}
				}
			})

			b.Run("ReadPassages/FileAndMarkdownExtraction", func(b *testing.B) {
				if len(c.candidates) == 0 {
					b.Fatal("no candidate passages to read")
				}
				reader, err := vaultreader.Readers{}.Open(c.vault)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					for _, candidate := range c.candidates {
						raw, err := reader.Read(ctx, candidate.Source)
						if err != nil {
							b.Fatal(err)
						}
						doc, err := markdown.Open(raw)
						if err != nil {
							b.Fatal(err)
						}
						_ = doc.Body()
					}
				}
			})
		})
	}
}
