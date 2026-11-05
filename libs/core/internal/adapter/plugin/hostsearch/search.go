// Package hostsearch lends the search index and the search floor of the vaults to the plugins that
// call the host.
package hostsearch

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/vector"
	indexport "github.com/jiva-studio/numen/libs/core/usecase/indexing/indexcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	searchpolicyport "github.com/jiva-studio/numen/libs/core/usecase/search/searchpolicycontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/usecase/vectors/embeddingcontract"
)

// Search is pluginport.PluginSearch over the index of the installation and the model that fills it.
type Search struct {
	// Embedder is the model that fills the index. Nil is an installation with no model, and every
	// call that needs one returns ErrNoModel.
	Embedder embeddingport.Embedder
	Samples  searchpolicyport.Samples
	Nearest  searchpolicyport.Nearest
	Progress indexport.IndexProgress
	// Stats reads what the index holds for GetIndexStats.
	Stats  *Stats
	Policy search.FloorPolicy
	// Finds is the search the window runs, over the same index and the model in Embedder.
	Finds PassageFinder
}

// PassageFinder is the search of the passages of a vault, which search.Search is.
type PassageFinder interface {
	Execute(ctx context.Context, v domain.Vault, query string, p search.Parameters) ([]domain.Passage, error)
}

// SearchPassages is the text of the vault that answers the query.
func (s Search) SearchPassages(ctx context.Context, v domain.Vault, q pluginport.PluginPassageQuery) ([]domain.Passage, error) {
	return nil, nil
}

// getRecipe is the recipe of the model in use, or ErrNoModel.
func (s Search) getRecipe() (string, error) {
	if s.Embedder == nil {
		return "", pluginport.ErrNoModel
	}
	return s.Embedder.GetModel().Recipe(), nil
}

// checkRecipe returns nil when the recipe is the one of the model in use.
func (s Search) checkRecipe(recipe string) error {
	current, err := s.getRecipe()
	if err != nil {
		return err
	}
	if recipe != current {
		return fmt.Errorf("%w: %q is not %q", pluginport.ErrRecipeMismatch, recipe, current)
	}
	return nil
}

// GetEmbeddingState reads the model in use and the progress of the vault under it from the index.
func (s Search) GetEmbeddingState(ctx context.Context, v domain.Vault) (pluginport.PluginEmbeddingState, error) {
	recipe, err := s.getRecipe()
	if err != nil {
		return pluginport.PluginEmbeddingState{}, err
	}
	held, embedded, err := s.Progress.GetProgress(ctx, v.ID, recipe)
	if err != nil {
		return pluginport.PluginEmbeddingState{}, fmt.Errorf("read the embedding progress of %s: %w", v.ID, err)
	}
	return pluginport.PluginEmbeddingState{
		Recipe:        recipe,
		Model:         s.Embedder.GetModel().Name,
		FloorHint:     s.Policy.Get(ctx, v, recipe).Hint,
		ChunkCount:    held,
		EmbeddedCount: embedded,
	}, nil
}

// GetIndexStats reads what the index holds for the vault and the space it takes, with the model in use.
func (s Search) GetIndexStats(ctx context.Context, v domain.Vault) (pluginport.PluginIndexStats, error) {
	recipe, err := s.getRecipe()
	if err != nil {
		return pluginport.PluginIndexStats{}, err
	}
	counts, sizes, err := s.Stats.Get(ctx, v.ID, recipe)
	if err != nil {
		return pluginport.PluginIndexStats{}, fmt.Errorf("read the index statistics of %s: %w", v.ID, err)
	}
	model := s.Embedder.GetModel()
	extensions := make([]pluginport.PluginExtensionStats, len(counts.Extensions))
	for i, one := range counts.Extensions {
		extensions[i] = pluginport.PluginExtensionStats{Extension: one.Extension, Sources: one.Sources, Chunks: one.Chunks}
	}
	return pluginport.PluginIndexStats{
		Recipe: recipe, Model: model.Name, Dimensions: int64(model.Dimensions),
		Chunks: counts.Chunks, Vectors: counts.Vectors,
		Sources: counts.Sources, Extensions: extensions,
		DatabaseBytes: sizes.Database, VectorBytes: sizes.Vectors,
	}, nil
}

// EmbedTexts embeds the texts with the model in use. More than MaxEmbeddedTexts texts, or a text
// longer than MaxEmbeddedBytes, returns ErrLimitExceeded.
func (s Search) EmbedTexts(ctx context.Context, texts []string) (pluginport.PluginEmbeddedTexts, error) {
	recipe, err := s.getRecipe()
	if err != nil {
		return pluginport.PluginEmbeddedTexts{}, err
	}
	if len(texts) > pluginport.MaxEmbeddedTexts {
		return pluginport.PluginEmbeddedTexts{}, fmt.Errorf("%w: %d texts, at most %d", pluginport.ErrLimitExceeded, len(texts), pluginport.MaxEmbeddedTexts)
	}
	for i, text := range texts {
		if len(text) > pluginport.MaxEmbeddedBytes {
			return pluginport.PluginEmbeddedTexts{}, fmt.Errorf("%w: text %d holds %d bytes, at most %d", pluginport.ErrLimitExceeded, i, len(text), pluginport.MaxEmbeddedBytes)
		}
	}
	if len(texts) == 0 {
		return pluginport.PluginEmbeddedTexts{Recipe: recipe}, nil
	}
	vectors, err := embeddingport.EmbedQueries(ctx, s.Embedder, texts)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return pluginport.PluginEmbeddedTexts{}, ctxErr
		}
		return pluginport.PluginEmbeddedTexts{}, fmt.Errorf("%w: %w", pluginport.ErrEmbeddingFailed, err)
	}
	if len(vectors) != len(texts) {
		return pluginport.PluginEmbeddedTexts{}, fmt.Errorf("%w: %d vectors for %d texts", pluginport.ErrEmbeddingFailed, len(vectors), len(texts))
	}
	if after := s.Embedder.GetModel().Recipe(); after != recipe {
		return pluginport.PluginEmbeddedTexts{}, fmt.Errorf("%w: the model changed while the texts were embedded", pluginport.ErrRecipeMismatch)
	}
	return pluginport.PluginEmbeddedTexts{Recipe: recipe, Vectors: vectors}, nil
}

// SampleSimilarities compares each query with the sampled small chunks of the vault. At most
// MaxSampledQueries queries are taken, else ErrLimitExceeded. A query whose dimension count differs
// from a passage's returns ErrInvalidQuery.
func (s Search) SampleSimilarities(ctx context.Context, v domain.Vault, recipe string, queries []pluginport.PluginSimilarityQuery) (pluginport.PluginSimilaritySample, error) {
	if err := s.checkRecipe(recipe); err != nil {
		return pluginport.PluginSimilaritySample{}, err
	}
	if len(queries) > pluginport.MaxSampledQueries {
		return pluginport.PluginSimilaritySample{}, fmt.Errorf("%w: %d queries, at most %d", pluginport.ErrLimitExceeded, len(queries), pluginport.MaxSampledQueries)
	}
	passages, err := s.Samples.ListSmallChunkVectors(ctx, v, recipe, pluginport.MaxSampledPassages)
	if err != nil {
		return pluginport.PluginSimilaritySample{}, err
	}
	sources := make(map[int64]struct{}, len(passages))
	for _, passage := range passages {
		sources[passage.Source] = struct{}{}
	}
	rows := make([][]float64, len(queries))
	for i, query := range queries {
		if err := ctx.Err(); err != nil {
			return pluginport.PluginSimilaritySample{}, err
		}
		row, err := compare(query, passages)
		if err != nil {
			return pluginport.PluginSimilaritySample{}, fmt.Errorf("query %d: %w", i, err)
		}
		rows[i] = row
	}
	return pluginport.PluginSimilaritySample{PassageCount: len(passages), SourceCount: len(sources), Rows: rows}, nil
}

// compare is the similarity of the query with each passage that belongs to another note than the
// query's, in the order of the passages.
func compare(query pluginport.PluginSimilarityQuery, passages []searchpolicyport.Passage) ([]float64, error) {
	if len(query.Vector) == 0 {
		return nil, nil
	}
	row := make([]float64, 0, len(passages))
	for _, passage := range passages {
		if query.Path != "" && passage.Path == query.Path {
			continue
		}
		if len(passage.Vector) != len(query.Vector) {
			return nil, fmt.Errorf("%w: a passage holds %d dimensions where the query has %d", pluginport.ErrInvalidQuery, len(passage.Vector), len(query.Vector))
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
func (s Search) GetSearchFloor(ctx context.Context, v domain.Vault, recipe string) (pluginport.PluginSearchFloor, error) {
	if err := s.checkRecipe(recipe); err != nil {
		return pluginport.PluginSearchFloor{}, err
	}
	got := s.Policy.Get(ctx, v, recipe)
	return pluginport.PluginSearchFloor{
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
			return fmt.Errorf("%w: %w", pluginport.ErrInvalidFloor, err)
		}
		return err
	}
	return nil
}

func convertSource(source search.FloorSourceKind) pluginport.PluginFloorSource {
	switch source {
	case search.FloorFromStore:
		return pluginport.PluginFloorStored
	case search.FloorFromModel:
		return pluginport.PluginFloorModelHint
	default:
		return pluginport.PluginFloorDefault
	}
}
