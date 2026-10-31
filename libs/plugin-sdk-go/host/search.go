package host

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/protocol"
)

var (
	// ErrPermissionDenied is what a search call gets from a plugin whose manifest does not declare the permission the call needs.
	ErrPermissionDenied = errors.New("host: the plugin does not hold the permission")
	// ErrNoModel is what a search call gets while the installation has no embedding model.
	ErrNoModel = errors.New("host: the installation has no embedding model")
	// ErrRecipeMismatch is what a call gets that names a recipe other than the one of the model in use.
	ErrRecipeMismatch = errors.New("host: the recipe is not the one of the model in use")
	// ErrInvalidFloor is what a floor gets that is not above zero and at most one.
	ErrInvalidFloor = errors.New("host: a floor is a cosine similarity above 0 and at most 1")
	// ErrLimitExceeded is what an embedding call gets that names more texts, or longer ones, than one call embeds.
	ErrLimitExceeded = errors.New("host: the call exceeds the limits of one embedding call")
	// ErrEmbeddingFailed is what an embedding call gets when the model could not answer.
	ErrEmbeddingFailed = errors.New("host: the embedding model could not answer")
)

// The permissions of the search calls. The reads, the embedding included, need PermissionIndexRead, and
// the floor needs PermissionSearchPolicy.
const (
	PermissionIndexRead    = "index:read"
	PermissionSearchPolicy = "search:policy"
)

// The limits the host applies to a search call. A list limit of zero, or above its maximum, is the
// maximum.
const (
	MaxListedTitles  = 1000
	MaxListedVectors = 1000
	MaxEmbeddedTexts = 256
	MaxEmbeddedBytes = 8192
)

// EmbeddingState is the model in use and how far a vault is embedded under it.
type EmbeddingState struct {
	// Recipe identifies the model in use. Every vector the host returns, and every floor it stores,
	// is kept under it.
	Recipe string
	// Model is the name of the model.
	Model string
	// FloorHint is the starting floor the plugin of the model reports, zero when it has none.
	FloorHint float64
	// ChunkCount is how many chunks of the vault can carry a vector, and EmbeddedCount how many do.
	ChunkCount    int64
	EmbeddedCount int64
	// IsReady is whether the vault holds chunks and every one of them carries a vector.
	IsReady bool
}

// EmbeddedTexts are the vectors the model in use made of texts.
type EmbeddedTexts struct {
	Recipe  string
	Vectors [][]float32
}

// NoteTitle is the title of a note and the source it belongs to in the index.
type NoteTitle struct {
	Source int64
	Title  string
}

// ChunkVector is the stored vector of a small chunk and the source it belongs to. Vector holds one
// signed byte per dimension.
type ChunkVector struct {
	Source int64
	Vector []byte
}

// FloorSource is where the floor a search applies comes from.
type FloorSource int

const (
	// FloorFromDefault: no other source answered.
	FloorFromDefault FloorSource = iota + 1
	// FloorFromModelHint: the starting floor the plugin of the embedding model reports.
	FloorFromModelHint
	// FloorFromStore: a plugin stored the floor for the vault and the recipe.
	FloorFromStore
)

// SearchFloor is the floor a search applies to a vault under a recipe, and the floor of each source
// it was chosen from. A source with no floor holds zero.
type SearchFloor struct {
	Floor   float64
	Source  FloorSource
	Stored  float64
	Hint    float64
	Default float64
}

// Search is the search index and the search floor of the vault the host shows, as a plugin works
// them. SearchClient works them through HostSearchService, and tests work them through
// hosttest.Search.
//
// A call that names a recipe other than the one of the model in use returns ErrRecipeMismatch, and a
// call while the installation has no model returns ErrNoModel. A call while the host shows no vault
// returns ErrNoVault.
//
// The reads need the index:read permission and the floor needs search:policy.
type Search interface {
	// With is the search of the vault of the identity. A call on it returns ErrUnknownVault when the
	// installation has no such vault.
	With(id VaultID) Search

	// GetEmbeddingState is the model in use and how far the vault is embedded under it. It is read
	// from the index at each call, so a plugin that starts after the vectors exist reads the same
	// state as one that was running when they were made.
	GetEmbeddingState(ctx context.Context) (EmbeddingState, error)
	// EmbedTexts embeds the texts with the model in use. At most MaxEmbeddedTexts texts of at most
	// MaxEmbeddedBytes bytes each are embedded in one call.
	EmbedTexts(ctx context.Context, texts []string) (EmbeddedTexts, error)
	// ListNoteTitles lists the titles of the notes of the vault in the order of their content hashes,
	// at most limit of them and at most MaxListedTitles.
	ListNoteTitles(ctx context.Context, limit int) ([]NoteTitle, error)
	// ListSmallChunkVectors lists the small chunks of the vault that carry a vector under the recipe,
	// in the order of their content hashes, at most limit of them and at most MaxListedVectors.
	ListSmallChunkVectors(ctx context.Context, recipe string, limit int) ([]ChunkVector, error)
	// GetBestSimilarity is the cosine similarity of the passage of the vault nearest the query vector
	// under the recipe, and zero where the vault has none.
	GetBestSimilarity(ctx context.Context, recipe string, query []float32) (float64, error)

	// GetSearchFloor is the floor a search applies to the vault under the recipe, and where it comes from.
	GetSearchFloor(ctx context.Context, recipe string) (SearchFloor, error)
	// SetSearchFloor stores the effective floor of the vault under the recipe. A floor of zero removes
	// the stored floor, and a floor above one or below zero returns ErrInvalidFloor. Setting the floor
	// that is stored changes nothing.
	SetSearchFloor(ctx context.Context, recipe string, floor float64) error
}

// ReadVectorsReadyEvent returns the identity of the vault that an event of the host names as having
// every chunk embedded, and false for any other event. The host emits the event after each indexing
// pass that leaves no vector pending, so it repeats: a handler reads the state again and acts only on
// what is not done.
func ReadVectorsReadyEvent(event string) (VaultID, bool) {
	vaultID, found := protocol.ReadVectorsReadyEvent(event)
	return VaultID(vaultID), found
}
