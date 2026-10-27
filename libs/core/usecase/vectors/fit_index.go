package vectors

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

// FitIndex sets the vector index to the model of the embedding provider: the
// width of its vectors, and on a rebuild only the vectors of its recipe.
type FitIndex struct {
	index embeddingport.VectorIndex

	// ShouldRebuildIndex removes the vectors of every other recipe.
	ShouldRebuildIndex bool
	// ErrorHandler, if set, receives a failed compaction. The removed rows stay
	// removed.
	ErrorHandler func(error)
}

// NewFitIndex returns the use case over the vector index given.
func NewFitIndex(index embeddingport.VectorIndex) FitIndex {
	return FitIndex{index: index}
}

// Execute fits the index to the embedder's model. Without an embedder the
// index is left as it is and search answers by words.
func (u FitIndex) Execute(ctx context.Context, embedder embeddingport.Embedder) error {
	if embedder == nil {
		return nil
	}
	model := embedder.GetModel()
	if err := u.index.FitVectors(ctx, model.Dimensions, model.Recipe()); err != nil {
		return err
	}
	if !u.ShouldRebuildIndex {
		return nil
	}
	if _, err := u.index.DeleteVectorsOfOtherRecipes(ctx, model.Recipe()); err != nil {
		return err
	}
	if err := u.index.Compact(ctx); err != nil && u.ErrorHandler != nil {
		u.ErrorHandler(err)
	}
	return nil
}
