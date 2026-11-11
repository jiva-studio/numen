// Package hostsearch lends the search index and the search floor of the vaults to the plugins that
// call the host.
package hostsearch

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchpolicy"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
)

// Embedder is the model that fills the index.
type Embedder interface {
	// GetModel is the identity every vector this embedder returns belongs to.
	GetModel() embedding.Model
}

// Samples reads the small chunks of a vault that carry a vector. The order is a function of the content
// hashes in the index, so the same index yields the same sample.
type Samples interface {
	// ListSmallChunkVectors returns at most `limit` small chunks of the vault that carry a
	// vector under the recipe.
	ListSmallChunkVectors(ctx context.Context, v domain.Vault, recipe string, limit int) ([]searchpolicy.Passage, error)
}

// Nearest reads the similarity of the best passage of a vault to a vector.
type Nearest interface {
	// GetBestSimilarity returns the cosine similarity of the passage of the vault nearest the query
	// vector under the recipe, and zero where there is none.
	GetBestSimilarity(ctx context.Context, vaultID domain.VaultID, recipe string, query []float32) (float64, error)
}

// Progress reads how far a vault is embedded.
type Progress interface {
	// GetProgress returns how many of the vault's chunks can have a vector, and how many of those
	// have one for the named model.
	GetProgress(ctx context.Context, vaultID domain.VaultID, model string) (held, embedded int64, err error)
}

// Search lends the search index and the search floors of the vaults to the plugins, over the index of
// the installation and the model that fills it. It implements the capabilities of
// hostendpoint.SearchAccess.
type Search struct {
	Embedder Embedder
	Queries  *vectorscase.EmbedQueries
	Samples  Samples
	Nearest  Nearest
	Progress Progress
	Stats    *Stats
	Policy   search.FloorPolicy
	Passages *search.SearchPassages
}

// SearchPassages returns passages matching the query by keywords and, when a model exists, by
// embedding.
func (s Search) SearchPassages(ctx context.Context, v domain.Vault, q plugin.PassageQuery) ([]domain.Passage, error) {
	return s.Passages.Execute(ctx, v, search.PassageQuery{
		Query: q.Query, Limit: q.Limit, Kinds: q.Kinds, PathPrefix: q.PathPrefix,
	})
}

func (s Search) getRecipe() (string, error) {
	if s.Embedder == nil {
		return "", plugin.ErrNoModel
	}
	return s.Embedder.GetModel().Recipe(), nil
}

func (s Search) checkRecipe(recipe string) error {
	current, err := s.getRecipe()
	if err != nil {
		return err
	}
	if recipe != current {
		return fmt.Errorf("%w: %q is not %q", plugin.ErrRecipeMismatch, recipe, current)
	}
	return nil
}

// GetEmbeddingState reads the model in use and the progress of the vault under it from the index.
func (s Search) GetEmbeddingState(ctx context.Context, v domain.Vault) (plugin.EmbeddingState, error) {
	recipe, err := s.getRecipe()
	if err != nil {
		return plugin.EmbeddingState{}, err
	}
	held, embedded, err := s.Progress.GetProgress(ctx, v.ID, recipe)
	if err != nil {
		return plugin.EmbeddingState{}, fmt.Errorf("read the embedding progress of %s: %w", v.ID, err)
	}
	return plugin.EmbeddingState{
		Recipe:        recipe,
		Model:         s.Embedder.GetModel().Name,
		FloorHint:     s.Policy.Get(ctx, v, recipe).Hint,
		ChunkCount:    held,
		EmbeddedCount: embedded,
	}, nil
}

// GetIndexStats reads what the index holds for the vault and the space it takes, with the model in use.
func (s Search) GetIndexStats(ctx context.Context, v domain.Vault) (plugin.IndexStats, error) {
	recipe, err := s.getRecipe()
	if err != nil {
		return plugin.IndexStats{}, err
	}
	counts, sizes, err := s.Stats.Get(ctx, v.ID, recipe)
	if err != nil {
		return plugin.IndexStats{}, fmt.Errorf("read the index statistics of %s: %w", v.ID, err)
	}
	model := s.Embedder.GetModel()
	extensions := make([]plugin.ExtensionStats, len(counts.Extensions))
	for i, one := range counts.Extensions {
		extensions[i] = plugin.ExtensionStats{Extension: one.Extension, Sources: one.Sources, Chunks: one.Chunks}
	}
	return plugin.IndexStats{
		Recipe: recipe, Model: model.Name, Dimensions: int64(model.Dimensions),
		Chunks: counts.Chunks, Vectors: counts.Vectors,
		Sources: counts.Sources, Extensions: extensions,
		DatabaseBytes: sizes.Database, VectorBytes: sizes.Vectors,
	}, nil
}

// EmbedTexts embeds the texts with the model in use. More than MaxEmbeddedTexts texts, or a text
// longer than MaxEmbeddedBytes, returns ErrLimitExceeded.
func (s Search) EmbedTexts(ctx context.Context, texts []string) (plugin.EmbeddedTexts, error) {
	recipe, err := s.getRecipe()
	if err != nil {
		return plugin.EmbeddedTexts{}, err
	}
	if len(texts) > plugin.MaxEmbeddedTexts {
		return plugin.EmbeddedTexts{}, fmt.Errorf("%w: %d texts, at most %d", plugin.ErrLimitExceeded, len(texts), plugin.MaxEmbeddedTexts)
	}
	for i, text := range texts {
		if len(text) > plugin.MaxEmbeddedBytes {
			return plugin.EmbeddedTexts{}, fmt.Errorf("%w: text %d holds %d bytes, at most %d", plugin.ErrLimitExceeded, i, len(text), plugin.MaxEmbeddedBytes)
		}
	}
	if len(texts) == 0 {
		return plugin.EmbeddedTexts{Recipe: recipe}, nil
	}
	vectors, err := s.Queries.Execute(ctx, texts)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return plugin.EmbeddedTexts{}, ctxErr
		}
		return plugin.EmbeddedTexts{}, fmt.Errorf("%w: %w", plugin.ErrEmbeddingFailed, err)
	}
	if len(vectors) != len(texts) {
		return plugin.EmbeddedTexts{}, fmt.Errorf("%w: %d vectors for %d texts", plugin.ErrEmbeddingFailed, len(vectors), len(texts))
	}
	if after := s.Embedder.GetModel().Recipe(); after != recipe {
		return plugin.EmbeddedTexts{}, fmt.Errorf("%w: the model changed while the texts were embedded", plugin.ErrRecipeMismatch)
	}
	return plugin.EmbeddedTexts{Recipe: recipe, Vectors: vectors}, nil
}

// SampleSimilarities compares each query with the sampled small chunks of the vault. At most
// MaxSampledQueries queries are taken, else ErrLimitExceeded. A query whose dimension count differs
// from a passage's returns ErrInvalidQuery.
func (s Search) SampleSimilarities(ctx context.Context, v domain.Vault, recipe string, queries []plugin.SimilarityQuery) (plugin.SimilaritySample, error) {
	if err := s.checkRecipe(recipe); err != nil {
		return plugin.SimilaritySample{}, err
	}
	if len(queries) > plugin.MaxSampledQueries {
		return plugin.SimilaritySample{}, fmt.Errorf("%w: %d queries, at most %d", plugin.ErrLimitExceeded, len(queries), plugin.MaxSampledQueries)
	}
	passages, err := s.Samples.ListSmallChunkVectors(ctx, v, recipe, plugin.MaxSampledPassages)
	if err != nil {
		return plugin.SimilaritySample{}, err
	}
	sources := make(map[int64]struct{}, len(passages))
	for _, passage := range passages {
		sources[passage.Source] = struct{}{}
	}
	rows := make([][]float64, len(queries))
	for i, query := range queries {
		if err := ctx.Err(); err != nil {
			return plugin.SimilaritySample{}, err
		}
		row, err := compare(query, passages)
		if err != nil {
			return plugin.SimilaritySample{}, fmt.Errorf("query %d: %w", i, err)
		}
		rows[i] = row
	}
	return plugin.SimilaritySample{PassageCount: len(passages), SourceCount: len(sources), Rows: rows}, nil
}

func compare(query plugin.SimilarityQuery, passages []searchpolicy.Passage) ([]float64, error) {
	if len(query.Vector) == 0 {
		return nil, nil
	}
	row := make([]float64, 0, len(passages))
	for _, passage := range passages {
		if query.Path != "" && passage.Path == query.Path {
			continue
		}
		if len(passage.Vector) != len(query.Vector) {
			return nil, fmt.Errorf("%w: a passage holds %d dimensions where the query has %d", plugin.ErrInvalidQuery, len(passage.Vector), len(query.Vector))
		}
		row = append(row, vector.SimilarityBytes(query.Vector, passage.Vector))
	}
	return row, nil
}

// GetBestSimilarity is the similarity of the passage of the vault nearest the query.
func (s Search) GetBestSimilarity(ctx context.Context, v domain.Vault, recipe string, query []float32) (float64, error) {
	if err := s.checkRecipe(recipe); err != nil {
		return 0, err
	}
	return s.Nearest.GetBestSimilarity(ctx, v.ID, recipe, query)
}

// GetSearchFloor is the floor a search applies to the vault under the recipe.
func (s Search) GetSearchFloor(ctx context.Context, v domain.Vault, recipe string) (plugin.SearchFloor, error) {
	if err := s.checkRecipe(recipe); err != nil {
		return plugin.SearchFloor{}, err
	}
	got := s.Policy.Get(ctx, v, recipe)
	return plugin.SearchFloor{
		Floor: got.Floor, Source: convertSource(got.Source),
		Stored: got.Stored, Hint: got.Hint, Default: got.Default,
	}, nil
}

// SetSearchFloor stores the floor of the vault under the recipe.
func (s Search) SetSearchFloor(ctx context.Context, v domain.Vault, recipe string, floor float64) error {
	if err := s.checkRecipe(recipe); err != nil {
		return err
	}
	if err := s.Policy.Set(ctx, v, recipe, floor); err != nil {
		if errors.Is(err, search.ErrInvalidFloor) {
			return fmt.Errorf("%w: %w", plugin.ErrInvalidFloor, err)
		}
		return err
	}
	return nil
}

func convertSource(source search.FloorSourceKind) plugin.FloorSource {
	switch source {
	case search.FloorFromStore:
		return plugin.FloorStored
	case search.FloorFromModel:
		return plugin.FloorModelHint
	default:
		return plugin.FloorDefault
	}
}

// Access returns the capabilities of the search the way hostendpoint serves them.
func (s Search) Access() *hostendpoint.SearchAccess {
	return &hostendpoint.SearchAccess{SearchIndex: s, PassageSearch: s, SearchFloors: s}
}
