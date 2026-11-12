package searchsourceport

import "context"

// VectorIndex is the store of vectors as a whole: its width and the recipes it
// holds vectors of.
type VectorIndex interface {
	// FitVectors makes the index hold vectors of the width given, filled from
	// the vectors already kept under the recipe.
	FitVectors(ctx context.Context, dims int, recipe string) error
	// DeleteVectorsOfOtherRecipes removes the vectors kept under any recipe but
	// the one given, and returns how many were removed.
	DeleteVectorsOfOtherRecipes(ctx context.Context, recipe string) (int64, error)
	// ApplyMinChunkLength sets the shortest small chunk that gets a vector, and removes
	// the vectors of the shorter ones. A shorter chunk is searched by its words only.
	// It returns how many vectors it removed.
	ApplyMinChunkLength(ctx context.Context, length int) (int64, error)
	// Compact returns the space of removed rows to the file system.
	Compact(ctx context.Context) error
}
