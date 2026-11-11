package searchpolicyport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchpolicy"
	"github.com/jiva-studio/numen/libs/note/domain"
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

// Samples reads the vectors that a plugin measures a vault with.
//
// The order is a function of the content hashes in the index, so the same index
// yields the same sample.
type Samples interface {
	// ListSmallChunkVectors returns at most `limit` small chunks of the vault that carry a
	// vector under the recipe.
	ListSmallChunkVectors(ctx context.Context, v domain.Vault, recipe string, limit int) ([]searchpolicy.Passage, error)
}
