package vectors

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
)

// vectorIndex records the calls made to it, in order.
type vectorIndex struct {
	calls      []string
	dims       int
	recipe     string
	compactErr error
	applied    int
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

func (x *vectorIndex) ApplyMinChunkLength(_ context.Context, length int) (int64, error) {
	x.calls = append(x.calls, "min "+strconv.Itoa(length))
	x.applied = length
	return 0, nil
}

func (x *vectorIndex) Compact(context.Context) error {
	x.calls = append(x.calls, "compact")
	return x.compactErr
}

func TestFitIndexSetsTheWidthOfTheModel(t *testing.T) {
	index := &vectorIndex{}
	model := &embedder{dims: 8}

	if err := NewFitIndex(index, nil).Execute(t.Context(), sourcefake.Wrap(model)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(index.calls, []string{"fit", "min 0"}) {
		t.Errorf("calls = %v, want [fit min 0]", index.calls)
	}
	if index.dims != 8 || index.recipe != model.Describe().Recipe() {
		t.Errorf("fit to %d %q", index.dims, index.recipe)
	}
}

func TestFitIndexWithoutEmbedderChangesNothing(t *testing.T) {
	index := &vectorIndex{}

	if err := NewFitIndex(index, nil).Execute(t.Context(), nil); err != nil {
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
	fit := NewFitIndex(index, nil)
	fit.ShouldRebuildIndex = true
	fit.errorHandler = func(err error) { reported = err }

	if err := fit.Execute(t.Context(), sourcefake.Wrap(model)); err != nil {
		t.Fatal(err)
	}
	want := []string{"fit", "min 0", "delete " + model.Describe().Recipe(), "compact"}
	if !slices.Equal(index.calls, want) {
		t.Errorf("calls = %v, want %v", index.calls, want)
	}
	if !errors.Is(reported, failed) {
		t.Errorf("reported %v, want %v", reported, failed)
	}
}

// The minimum the model reports is applied to the index, so the chunks shorter than it lose their
// vectors, and it is applied after the index is fitted to the model.
func TestFitIndexAppliesTheMinimumChunkLengthOfTheModel(t *testing.T) {
	index := &vectorIndex{}
	model := &embedder{dims: 8, minLen: 40}

	if err := NewFitIndex(index, nil).Execute(t.Context(), sourcefake.Wrap(model)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(index.calls, []string{"fit", "min 40"}) {
		t.Errorf("calls = %v, want [fit min 40]", index.calls)
	}
}

// The minimum is not part of the recipe: changing it must not make a stored vector stale.
func TestMinimumChunkLengthIsNotPartOfTheRecipe(t *testing.T) {
	a := (&embedder{dims: 8, minLen: 0}).Describe().Recipe()
	b := (&embedder{dims: 8, minLen: 500}).Describe().Recipe()
	if a != b {
		t.Errorf("recipe changed with the minimum: %q and %q", a, b)
	}
}
