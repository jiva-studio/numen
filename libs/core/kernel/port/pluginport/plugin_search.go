package pluginport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// The limits of the search index calls.
const (
	// MaxSampledQueries is how many queries one SampleSimilarities call takes, and MaxSampledPassages how
	// many small chunks it compares each of them with.
	MaxSampledQueries  = 256
	MaxSampledPassages = 500
)

// PluginEmbeddingState is the model in use and how far a vault is embedded under it.
type PluginEmbeddingState struct {
	Recipe        string
	Model         string
	FloorHint     float64
	ChunkCount    int64
	EmbeddedCount int64
}

// IsReady reports whether the vault holds chunks and every one of them carries a vector.
func (s PluginEmbeddingState) IsReady() bool {
	return s.ChunkCount > 0 && s.EmbeddedCount >= s.ChunkCount
}

// PluginEmbeddedTexts are the vectors the model in use made of texts, and its recipe.
type PluginEmbeddedTexts struct {
	Recipe  string
	Vectors [][]float32
}

// PluginSimilarityQuery is a query vector, and the path of the note it was made from. The passages of
// that note are left out of its sample; an empty path leaves out none.
type PluginSimilarityQuery struct {
	Path   string
	Vector []float32
}

// PluginSimilaritySample is the cosine similarity of each query with the sampled passages of the vault.
type PluginSimilaritySample struct {
	PassageCount int
	SourceCount  int
	Rows         [][]float64
}

// PluginFloorSource is where the floor a search applies comes from.
type PluginFloorSource int

const (
	// PluginFloorDefault: no other source answered.
	PluginFloorDefault PluginFloorSource = iota + 1
	// PluginFloorModelHint: the starting floor the plugin of the embedding model reports.
	PluginFloorModelHint
	// PluginFloorStored: a plugin stored the floor for the vault and the recipe.
	PluginFloorStored
)

// PluginSearchFloor is the floor a search applies to a vault under a recipe, and the sources it was chosen from.
type PluginSearchFloor struct {
	Floor   float64
	Source  PluginFloorSource
	Stored  float64
	Hint    float64
	Default float64
}

// PluginSearchIndex is the search index of the vaults, as the plugins that call the host read it. Each
// call works the vault it is given. A call that names a recipe other than the one of the model in use
// returns plugin.ErrRecipeMismatch, and one on an installation with no model returns plugin.ErrNoModel.
type PluginSearchIndex interface {
	// GetEmbeddingState is the model in use and how far the vault is embedded under it.
	GetEmbeddingState(ctx context.Context, v domain.Vault) (PluginEmbeddingState, error)
	// GetIndexStats is what the index of the vault holds, with the model in use. The counts are at most
	// one second old and the sizes at most half a minute old.
	GetIndexStats(ctx context.Context, v domain.Vault) (PluginIndexStats, error)
	// EmbedTexts embeds the texts with the model in use.
	EmbedTexts(ctx context.Context, texts []string) (PluginEmbeddedTexts, error)
	// SampleSimilarities is the cosine similarity of each query with the small chunks of the vault that
	// carry a vector under the recipe and belong to a source other than the query's note. At most
	// MaxSampledQueries queries are taken, and at most MaxSampledPassages small chunks, in the order of
	// their content hashes. A query longer or shorter than the vectors of the index returns
	// plugin.ErrInvalidQuery.
	SampleSimilarities(ctx context.Context, v domain.Vault, recipe string, queries []PluginSimilarityQuery) (PluginSimilaritySample, error)
	// GetBestSimilarity is the cosine similarity of the passage of the vault nearest the query vector,
	// zero where there is none.
	GetBestSimilarity(ctx context.Context, v domain.Vault, recipe string, query []float32) (float64, error)
}

// PluginPassageSearch searches the text of the vaults for the plugins that call the host.
type PluginPassageSearch interface {
	// SearchPassages returns the passages of the vault matching the query, best first, from every
	// source and every text version: by keywords, and by embedding when the installation has a model.
	// An installation with no model searches by keywords only.
	SearchPassages(ctx context.Context, v domain.Vault, q PluginPassageQuery) ([]domain.Passage, error)
}

// PluginPassageQuery is a passage search request.
type PluginPassageQuery struct {
	Query      string
	Limit      int
	Kinds      []domain.SourceKind
	PathPrefix string
}

// PluginSearchPolicy is the floor a search by meaning applies to the vaults, as the plugins that call
// the host read and set it. The recipe rules of PluginSearchIndex apply.
type PluginSearchPolicy interface {
	// GetSearchFloor is the floor a search applies to the vault under the recipe.
	GetSearchFloor(ctx context.Context, v domain.Vault, recipe string) (PluginSearchFloor, error)
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

// PluginExtensionStats is the sources of one file extension and the chunks they hold.
type PluginExtensionStats struct {
	Extension string
	Sources   int64
	Chunks    int64
}

// PluginIndexStats is what the index of a vault holds, and the space it takes.
type PluginIndexStats struct {
	Recipe        string
	Model         string
	Dimensions    int64
	Chunks        int64
	Vectors       int64
	Sources       int64
	Extensions    []PluginExtensionStats
	DatabaseBytes int64
	VectorBytes   int64
}
