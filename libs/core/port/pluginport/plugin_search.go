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
)

// The limits of the search index calls. A list limit of zero, or above its maximum, is the maximum.
const (
	MaxListedTitles  = 1000
	MaxListedVectors = 1000
	MaxEmbeddedTexts = 256
	MaxEmbeddedBytes = 8192
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

// PluginNoteTitle is the title of a note and the source it belongs to.
type PluginNoteTitle struct {
	Source int64
	Title  string
}

// PluginChunkVector is the stored vector of a small chunk and the source it belongs to.
type PluginChunkVector struct {
	Source int64
	Vector []byte
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
	// EmbedTexts embeds the texts with the model in use.
	EmbedTexts(ctx context.Context, texts []string) (PluginEmbeddedTexts, error)
	// ListNoteTitles lists at most limit note titles of the vault, in the order of their content hashes.
	ListNoteTitles(ctx context.Context, v domain.Vault, limit int) ([]PluginNoteTitle, error)
	// ListSmallChunkVectors lists at most limit small chunks of the vault that carry a vector under the
	// recipe, in the order of their content hashes.
	ListSmallChunkVectors(ctx context.Context, v domain.Vault, recipe string, limit int) ([]PluginChunkVector, error)
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
