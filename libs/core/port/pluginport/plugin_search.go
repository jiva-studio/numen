package pluginport

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

var (
	// ErrNoModel is a search call on an installation that has no embedding model.
	ErrNoModel = errors.New("the installation has no embedding model")
	// ErrRecipeMismatch is a search call that names a recipe other than the one of the model in use.
	ErrRecipeMismatch = errors.New("the recipe is not the one of the model in use")
	// ErrInvalidFloor is a floor outside the range above zero and up to one.
	ErrInvalidFloor = errors.New("a floor is a cosine similarity above 0 and at most 1")
	// ErrLimitExceeded is an embedding call with more texts, or longer ones, than one call embeds.
	ErrLimitExceeded = errors.New("the call exceeds the limits of one embedding call")
	// ErrEmbeddingFailed is an embedding model that could not answer.
	ErrEmbeddingFailed = errors.New("the embedding model could not answer")
	// ErrInvalidQuery is a query vector whose dimension count is not the one of the vectors in the index.
	ErrInvalidQuery = errors.New("a query vector does not match the vectors of the index")
)

// The limits of the search index calls.
const (
	MaxEmbeddedTexts = 256
	MaxEmbeddedBytes = 8192
	// MaxSampledQueries is how many queries one SampleSimilarities call takes, and MaxSampledPassages how
	// many small chunks it compares each of them with.
	MaxSampledQueries  = 256
	MaxSampledPassages = 500
)

// PluginEmbeddingState is the model in use and how far a vault is embedded under it.
type PluginEmbeddingState struct {
	Recipe string
	Model  string
	// FloorHint is the starting floor the plugin of the model reports, zero when it has none.
	FloorHint float64
	// ChunkCount is how many chunks of the vault can carry a vector, and EmbeddedCount how many do.
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
	// PassageCount is how many passages were sampled, and SourceCount how many sources they come from.
	PassageCount int
	SourceCount  int
	// Rows holds, for each query in the order given, its similarity with each sampled passage of another
	// source. A query with no vector has an empty row.
	Rows [][]float64
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
	Floor  float64
	Source PluginFloorSource
	// Stored, Hint and Default are the floor of each source, zero where the source has none.
	Stored  float64
	Hint    float64
	Default float64
}

// PluginSearchIndex is the search index of the vaults, as the plugins that call the host read it. Each
// call works the vault it is given. A call that names a recipe other than the one of the model in use
// returns ErrRecipeMismatch, and one on an installation with no model returns ErrNoModel.
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
	// ErrInvalidQuery.
	SampleSimilarities(ctx context.Context, v domain.Vault, recipe string, queries []PluginSimilarityQuery) (PluginSimilaritySample, error)
	// GetBestSimilarity is the cosine similarity of the passage of the vault nearest the query vector,
	// zero where there is none.
	GetBestSimilarity(ctx context.Context, v domain.Vault, recipe string, query []float32) (float64, error)
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

// PluginSearch is the search index and the search floor of the vaults.
type PluginSearch interface {
	PluginSearchIndex
	PluginSearchPolicy
}

// PluginSearchHost receives the search index that plugins work on.
type PluginSearchHost interface {
	// SetPluginSearch is the search index the plugins work. Nil is none.
	SetPluginSearch(search PluginSearch)
}

// PluginExtensionStats is the sources of one file extension and the chunks they hold.
type PluginExtensionStats struct {
	// Extension is in lower case, without the dot, and empty for a source with none.
	Extension string
	Sources   int64
	Chunks    int64
}

// PluginIndexStats is what the index of a vault holds, and the space it takes.
type PluginIndexStats struct {
	// Recipe, Model and Dimensions identify the model in use.
	Recipe     string
	Model      string
	Dimensions int64
	// Chunks is how many chunks can carry a vector, and Vectors how many of them do.
	Chunks  int64
	Vectors int64
	// Sources is the notes and files of the vault, and Extensions splits them by file extension, the
	// extension with the most sources first.
	Sources    int64
	Extensions []PluginExtensionStats
	// DatabaseBytes is the size of the index database on disk, and VectorBytes the size of the tables
	// that hold vectors.
	DatabaseBytes int64
	VectorBytes   int64
}
