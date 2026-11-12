package vectors

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
)

// FitIndex sets the vector index to the model of the embedding provider: the
// width of its vectors, and on a rebuild only the vectors of its recipe.
type FitIndex struct {
	index embeddingport.VectorIndex

	ShouldRebuildIndex bool
	errorHandler       func(error)
}

// NewFitIndex returns the use case over the vector index given. A nil errorHandler ignores the error of the
// compaction after a rebuild.
func NewFitIndex(index embeddingport.VectorIndex, errorHandler func(error)) FitIndex {
	return FitIndex{index: index, errorHandler: errorHandler}
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
	if _, err := u.index.ApplyMinChunkLength(ctx, model.MinChunkLength); err != nil {
		return err
	}
	if !u.ShouldRebuildIndex {
		return nil
	}
	if _, err := u.index.DeleteVectorsOfOtherRecipes(ctx, model.Recipe()); err != nil {
		return err
	}
	if err := u.index.Compact(ctx); err != nil && u.errorHandler != nil {
		u.errorHandler(err)
	}
	return nil
}
