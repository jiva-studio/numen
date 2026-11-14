// Package hostsearch lends the search index and the search floor of the vaults to the plugins that
// call the host.
package hostsearch

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchpolicy"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
)

// Source is the search source that fills the index.
type Source interface {
	// Describe is the description every vector this source returns belongs to.
	Describe() searchsource.Description
}

// Samples reads the small chunks of a vault that carry a vector. The order is a function of the content
// hashes in the index, so the same index yields the same sample.
type Samples interface {
	// ListSmallChunkVectors returns at most `limit` small chunks of the vault that carry a
	// vector under the recipe.
	ListSmallChunkVectors(ctx context.Context, v domain.Vault, recipe string, limit int) ([]searchpolicy.SampledChunk, error)
}

// Nearest reads the similarity of the best chunk of a vault to a vector.
type Nearest interface {
	// Execute returns the cosine similarity of the chunk of the vault nearest the query vector
	// under the recipe, and zero where there is none.
	Execute(ctx context.Context, vaultID domain.VaultID, recipe string, query []float32) (float64, error)
}

// Progress reads how far a vault is embedded.
type Progress interface {
	// GetProgress returns how many of the vault's chunks can have a vector, and how many of those
	// have one for the named model.
	GetProgress(ctx context.Context, vaultID domain.VaultID, model string) (held, embedded int64, err error)
}

// Search lends the search index and the search floors of the vaults to the plugins, over the index of
// the installation and the search source that fills it. It implements the capabilities of
// hostendpoint.SearchAccess.
type Search struct {
	Source     Source
	Queries    *vectorscase.EmbedQueries
	Samples    Samples
	Nearest    Nearest
	Places     search.FindNearest
	Progress   Progress
	Stats      *search.Stats
	Policy     search.FloorPolicy
	TextSearch *search.SearchText
}

// SearchText returns search results matching the query by keywords and, when a search source exists, by
// the source.
func (s Search) SearchText(ctx context.Context, v domain.Vault, q plugin.SearchResultQuery) ([]domain.SearchResult, error) {
	return s.TextSearch.Execute(ctx, v, search.SearchResultQuery{
		Query: q.Query, Limit: q.Limit, Kinds: q.Kinds, PathPrefix: q.PathPrefix,
	})
}

func (s Search) describe() (searchsource.Description, error) {
	if s.Source == nil {
		return searchsource.Description{}, plugin.ErrNoModel
	}
	described := s.Source.Describe()
	if !described.IsDescribed() {
		return searchsource.Description{}, plugin.ErrNoModel
	}
	return described, nil
}

func (s Search) getRecipe() (string, error) {
	described, err := s.describe()
	if err != nil {
		return "", err
	}
	return described.Recipe(), nil
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

// GetEmbeddingState reads the search source in use and the progress of the vault under it from the index.
func (s Search) GetEmbeddingState(ctx context.Context, v domain.Vault) (plugin.EmbeddingState, error) {
	described, err := s.describe()
	if err != nil {
		return plugin.EmbeddingState{}, err
	}
	return s.Policy.GetEmbeddingState(ctx, v, s.Progress, described)
}

// GetIndexStats reads what the index holds for the vault and the space it takes, with the search source in use.
func (s Search) GetIndexStats(ctx context.Context, v domain.Vault) (plugin.IndexStats, error) {
	described, err := s.describe()
	if err != nil {
		return plugin.IndexStats{}, err
	}
	return s.Stats.Execute(ctx, v, described)
}

// EmbedTexts embeds the texts as queries of the search source in use. More than MaxEmbeddedTexts texts, or
// a text longer than MaxEmbeddedBytes, returns ErrLimitExceeded.
func (s Search) EmbedTexts(ctx context.Context, texts []string) (plugin.EmbeddedTexts, error) {
	if _, err := s.describe(); err != nil || s.Queries == nil {
		return plugin.EmbeddedTexts{}, plugin.ErrNoModel
	}
	return s.Queries.EmbedTexts(ctx, s.Source.Describe, texts)
}

// SampleSimilarities compares each query with the sampled small chunks of the vault. At most
// MaxSampledQueries queries are taken, else ErrLimitExceeded. A query whose dimension count differs
// from a chunk's returns ErrInvalidQuery.
func (s Search) SampleSimilarities(ctx context.Context, v domain.Vault, recipe string, queries []plugin.SimilarityQuery) (plugin.SimilaritySample, error) {
	if err := s.checkRecipe(recipe); err != nil {
		return plugin.SimilaritySample{}, err
	}
	return search.SampleSimilarities(ctx, s.Samples, v, recipe, queries)
}

// GetBestSimilarity is the similarity of the chunk of the vault nearest the query.
func (s Search) GetBestSimilarity(ctx context.Context, v domain.Vault, recipe string, query []float32) (float64, error) {
	if err := s.checkRecipe(recipe); err != nil {
		return 0, err
	}
	return s.Nearest.Execute(ctx, v.ID, recipe, query)
}

// FindNearestChunks returns the small chunks of the vault nearest the vector under the recipe of the place,
// nearest first, at or above the floor the query names, else the effective floor of the vault.
func (s Search) FindNearestChunks(ctx context.Context, v domain.Vault, q sourcedomain.NearestQuery) ([]searchsource.Match, error) {
	if err := s.checkRecipe(q.Recipe); err != nil {
		return nil, err
	}
	if q.Limit <= 0 {
		return nil, nil
	}
	if q.Floor <= 0 {
		q.Floor = s.Policy.GetFloor(ctx, v, q.Recipe)
	}
	return s.Places.Execute(ctx, v.ID, q)
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
	return &hostendpoint.SearchAccess{SearchIndex: s, TextSearch: s, SearchFloors: s}
}
