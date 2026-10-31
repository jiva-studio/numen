package searchpolicyport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// FloorStore keeps the effective similarity floor of each vault under each recipe.
type FloorStore interface {
	// ReadFloor returns the floor kept for the vault under the recipe, and false where none is kept.
	ReadFloor(ctx context.Context, v domain.Vault, recipe string) (float64, bool, error)
	// WriteFloor keeps the floor, replacing the one kept.
	WriteFloor(ctx context.Context, v domain.Vault, recipe string, floor float64) error
	// DeleteFloor removes the floor kept. A floor that is not kept is removed already.
	DeleteFloor(ctx context.Context, v domain.Vault, recipe string) error
}

// Title is the name of a note and the source it belongs to.
type Title struct {
	Source int64
	Text   string
}

// Passage is a small chunk that carries a vector, and the source it belongs to.
// Vector is the stored form of the vector.
type Passage struct {
	Source int64
	Vector []byte
}

// Samples reads the titles and vectors that a plugin measures a vault with.
//
// The order of both is a function of the content hashes in the index, so the
// same index yields the same samples.
type Samples interface {
	// ListNoteTitles returns at most `limit` note titles of the vault.
	ListNoteTitles(ctx context.Context, v domain.Vault, limit int) ([]Title, error)
	// ListSmallChunkVectors returns at most `limit` small chunks of the vault that carry a
	// vector under the recipe.
	ListSmallChunkVectors(ctx context.Context, v domain.Vault, recipe string, limit int) ([]Passage, error)
}

// Nearest reads the similarity of the best passage of a vault to a vector.
type Nearest interface {
	// GetBestSimilarity returns the cosine similarity of the passage of the vault
	// nearest the query vector under the recipe, and zero where there is none.
	GetBestSimilarity(ctx context.Context, vaultID domain.VaultID, recipe string, query []float32) (float64, error)
}
