package pluginport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// PluginSearchIndex is the search index of the vaults, as the plugins that call the host read it. Each
// call works the vault it is given. A call that names a recipe other than the one of the model in use
// returns plugin.ErrRecipeMismatch, and one on an installation with no model returns plugin.ErrNoModel.
type PluginSearchIndex interface {
	// GetEmbeddingState is the model in use and how far the vault is embedded under it.
	GetEmbeddingState(ctx context.Context, v domain.Vault) (plugin.EmbeddingState, error)
	// GetIndexStats is what the index of the vault holds, with the model in use. The counts are at most
	// one second old and the sizes at most half a minute old.
	GetIndexStats(ctx context.Context, v domain.Vault) (plugin.IndexStats, error)
	// EmbedTexts embeds the texts with the model in use.
	EmbedTexts(ctx context.Context, texts []string) (plugin.EmbeddedTexts, error)
	// SampleSimilarities is the cosine similarity of each query with the small chunks of the vault that
	// carry a vector under the recipe and belong to a source other than the query's note. At most
	// MaxSampledQueries queries are taken, and at most MaxSampledPassages small chunks, in the order of
	// their content hashes. A query longer or shorter than the vectors of the index returns
	// plugin.ErrInvalidQuery.
	SampleSimilarities(ctx context.Context, v domain.Vault, recipe string, queries []plugin.SimilarityQuery) (plugin.SimilaritySample, error)
	// GetBestSimilarity is the cosine similarity of the passage of the vault nearest the query vector,
	// zero where there is none.
	GetBestSimilarity(ctx context.Context, v domain.Vault, recipe string, query []float32) (float64, error)
}

// PluginPassageSearch searches the text of the vaults for the plugins that call the host.
type PluginPassageSearch interface {
	// SearchPassages returns the passages of the vault matching the query, best first, from every
	// source and every text version: by keywords, and by embedding when the installation has a model.
	// An installation with no model searches by keywords only.
	SearchPassages(ctx context.Context, v domain.Vault, q plugin.PassageQuery) ([]domain.Passage, error)
}

// PluginSearchPolicy is the floor a search by meaning applies to the vaults, as the plugins that call
// the host read and set it. The recipe rules of PluginSearchIndex apply.
type PluginSearchPolicy interface {
	// GetSearchFloor is the floor a search applies to the vault under the recipe.
	GetSearchFloor(ctx context.Context, v domain.Vault, recipe string) (plugin.SearchFloor, error)
	// SetSearchFloor stores the effective floor of the vault under the recipe, and removes it when the
	// floor is zero.
	SetSearchFloor(ctx context.Context, v domain.Vault, recipe string, floor float64) error
}

// PluginSearch is the search index, the passage search and the similarity threshold of the vaults.
type PluginSearch interface {
	PluginSearchIndex
	PluginPassageSearch
	PluginSearchPolicy
}

// PluginSearchHost receives the search index that plugins work on.
type PluginSearchHost interface {
	// SetPluginSearch is the search index the plugins work. Nil is none.
	SetPluginSearch(search PluginSearch)
}

