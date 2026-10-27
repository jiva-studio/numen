package vectors

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// vectorIndex records the calls made to it, in order.
type vectorIndex struct {
	calls      []string
	dims       int
	recipe     string
	compactErr error
}

func (x *vectorIndex) FitVectors(_ context.Context, dims int, recipe string) error {
	x.calls = append(x.calls, "fit")
	x.dims, x.recipe = dims, recipe
	return nil
}

func (x *vectorIndex) DeleteVectorsOfOtherRecipes(_ context.Context, recipe string) (int64, error) {
	x.calls = append(x.calls, "delete "+recipe)
	return 0, nil
}

func (x *vectorIndex) Compact(context.Context) error {
	x.calls = append(x.calls, "compact")
	return x.compactErr
}

func TestFitIndexSetsTheWidthOfTheModel(t *testing.T) {
	index := &vectorIndex{}
	model := &embedder{dims: 8}

	if err := NewFitIndex(index).Execute(t.Context(), model); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(index.calls, []string{"fit"}) {
		t.Errorf("calls = %v, want [fit]", index.calls)
	}
	if index.dims != 8 || index.recipe != model.GetModel().Recipe() {
		t.Errorf("fit to %d %q", index.dims, index.recipe)
	}
}

func TestFitIndexWithoutEmbedderChangesNothing(t *testing.T) {
	index := &vectorIndex{}

	if err := NewFitIndex(index).Execute(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if len(index.calls) != 0 {
		t.Errorf("calls = %v, want none", index.calls)
	}
}

func TestFitIndexOnRebuildDeletesOtherRecipesAndReportsCompaction(t *testing.T) {
	failed := errors.New("disk full")
	index := &vectorIndex{compactErr: failed}
	model := &embedder{dims: 8}
	var reported error
	fit := NewFitIndex(index)
	fit.ShouldRebuildIndex = true
	fit.ErrorHandler = func(err error) { reported = err }

	if err := fit.Execute(t.Context(), model); err != nil {
		t.Fatal(err)
	}
	want := []string{"fit", "delete " + model.GetModel().Recipe(), "compact"}
	if !slices.Equal(index.calls, want) {
		t.Errorf("calls = %v, want %v", index.calls, want)
	}
	if !errors.Is(reported, failed) {
		t.Errorf("reported %v, want %v", reported, failed)
	}
}
