package vectors

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

// FitIndex sets the vector index to the search source: the width of its
// vectors, and on a rebuild only the vectors of its recipe.
type FitIndex struct {
	index searchsourceport.VectorIndex

	shouldRebuildIndex bool
	errorHandler       func(error)
}

// NewFitIndex returns the use case over the vector index given. A nil errorHandler ignores the error of the
// compaction after a rebuild. With shouldRebuildIndex the vectors of other recipes are deleted.
func NewFitIndex(index searchsourceport.VectorIndex, errorHandler func(error), shouldRebuildIndex bool) FitIndex {
	if errorHandler == nil {
		errorHandler = func(error) {}
	}
	return FitIndex{index: index, errorHandler: errorHandler, shouldRebuildIndex: shouldRebuildIndex}
}

// Execute fits the index to the description of the source. Without a source, or with one that has not
// described itself, the index is left as it is and search answers by words.
func (u FitIndex) Execute(ctx context.Context, source searchsourceport.Source) error {
	if source == nil {
		return nil
	}
	described := source.Describe()
	if !described.IsDescribed() {
		return nil
	}
	if err := u.index.FitVectors(ctx, described.Width, described.Recipe()); err != nil {
		return err
	}
	if _, err := u.index.ApplyMinChunkLength(ctx, described.MinTextLength); err != nil {
		return err
	}
	if !u.shouldRebuildIndex {
		return nil
	}
	if _, err := u.index.DeleteVectorsOfOtherRecipes(ctx, described.Recipe()); err != nil {
		return err
	}
	if err := u.index.Compact(ctx); err != nil {
		u.errorHandler(err)
	}
	return nil
}
