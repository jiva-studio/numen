// Package fitindex sets the vector index to the width and recipe of the search source.
package fitindex

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

// Fit sets the vector index to the search source: the width of its
// vectors, and on a rebuild only the vectors of its recipe.
type Fit struct {
	index VectorIndex

	shouldRebuildIndex bool
	handleError        func(error)
}

// New returns the use case over the vector index given. A nil errorHandler ignores the error of the
// compaction after a rebuild. With shouldRebuildIndex the vectors of other recipes are deleted.
func New(index VectorIndex, errorHandler func(error), shouldRebuildIndex bool) Fit {
	if errorHandler == nil {
		errorHandler = func(error) {}
	}
	return Fit{index: index, handleError: errorHandler, shouldRebuildIndex: shouldRebuildIndex}
}

// Execute fits the index to the description of the source. Without a source, or with one that has not
// described itself, the index is left as it is and search answers by words.
func (u Fit) Execute(ctx context.Context, source searchsourceport.Source) error {
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
		u.handleError(err)
	}
	return nil
}
