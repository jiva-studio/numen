package hosttest_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

const searchRecipe = "local|bge|1024|512|mean|int8@0.4"

func newSearch() *hosttest.Search {
	search := hosttest.NewSearch("physics", "chemistry")
	search.SetModel(hosttest.SearchModel{Name: "bge", Recipe: searchRecipe, FloorHint: 0.4})
	search.SetEmbedder(func(text string) []float32 { return []float32{float32(len(text)), 1} })
	return search
}

func passagesOf(n int) []hosttest.SearchPassage {
	out := make([]hosttest.SearchPassage, n)
	for i := range out {
		out[i] = hosttest.SearchPassage{Path: fmt.Sprintf("note-%d.md", i+1), Vector: []float32{float32(n - i), float32(i + 1)}}
	}
	return out
}

func TestTheSearchRefusesACallWithoutThePermission(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.Deny(host.PermissionIndexRead)

	if _, err := search.GetEmbeddingState(ctx); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("state gave %v", err)
	}
	if _, err := search.EmbedTexts(ctx, []string{"a"}); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("embedding gave %v", err)
	}
	if _, err := search.SampleSimilarities(ctx, searchRecipe, nil); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("sample gave %v", err)
	}
	if _, err := search.GetBestSimilarity(ctx, searchRecipe, nil); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("best similarity gave %v", err)
	}
	if _, err := search.GetSearchFloor(ctx, searchRecipe); err != nil {
		t.Errorf("the floor needs another permission, and gave %v", err)
	}

	search.Deny(host.PermissionSearchPolicy)
	if _, err := search.GetSearchFloor(ctx, searchRecipe); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("floor read gave %v", err)
	}
	if err := search.SetSearchFloor(ctx, searchRecipe, 0.5); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("floor set gave %v", err)
	}
}

func TestTheSearchOfAnUnknownVaultIsRefusedAndAKnownOtherVaultIsItsOwn(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.SetPassages(searchRecipe, passagesOf(3))

	unknown := search.With("biology")
	if _, err := unknown.GetEmbeddingState(ctx); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("state gave %v", err)
	}
	if _, err := unknown.SampleSimilarities(ctx, searchRecipe, nil); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("sample gave %v", err)
	}
	if err := unknown.SetSearchFloor(ctx, searchRecipe, 0.5); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("floor set gave %v", err)
	}

	sample, err := search.With("chemistry").SampleSimilarities(ctx, searchRecipe, []host.SimilarityQuery{{Vector: []float32{1, 0}}})
	if err != nil || sample.PassageCount != 0 {
		t.Errorf("another vault sampled %+v (%v), want none", sample, err)
	}
	if err := search.With("chemistry").SetSearchFloor(ctx, searchRecipe, 0.6); err != nil {
		t.Fatal(err)
	}
	if _, found := search.StoredFloor(searchRecipe); found {
		t.Error("a floor set on another vault was stored for the shown one")
	}
}

func TestTheSearchOfAHostShowingNoVaultReturnsNoVault(t *testing.T) {
	search := newSearch()
	search.HideVault()
	if _, err := search.GetEmbeddingState(t.Context()); !errors.Is(err, host.ErrNoVault) {
		t.Errorf("state gave %v", err)
	}
	if _, err := search.With("chemistry").GetEmbeddingState(t.Context()); err != nil {
		t.Errorf("a vault named by identity gave %v", err)
	}
}

func TestTheSearchCapsASampleAndItsQueries(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.SetPassages(searchRecipe, passagesOf(host.MaxSampledPassages+50))

	sample, err := search.SampleSimilarities(ctx, searchRecipe, []host.SimilarityQuery{{Vector: []float32{1, 0}}})
	if err != nil || sample.PassageCount != host.MaxSampledPassages || len(sample.Rows[0]) != host.MaxSampledPassages {
		t.Errorf("sampled %d passages and %d similarities (%v), want %d", sample.PassageCount, len(sample.Rows[0]), err, host.MaxSampledPassages)
	}
	if _, err := search.SampleSimilarities(ctx, searchRecipe, make([]host.SimilarityQuery, host.MaxSampledQueries+1)); !errors.Is(err, host.ErrLimitExceeded) {
		t.Errorf("too many queries gave %v", err)
	}
}

func TestTheSearchSamplesInTheOrderOfTheIndexEveryTime(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.SetPassages(searchRecipe, passagesOf(30))
	queries := []host.SimilarityQuery{{Path: "note-3.md", Vector: []float32{1, 0}}, {Vector: []float32{0, 1}}}

	first, err := search.SampleSimilarities(ctx, searchRecipe, queries)
	if err != nil {
		t.Fatal(err)
	}
	second, err := search.SampleSimilarities(ctx, searchRecipe, queries)
	if err != nil {
		t.Fatal(err)
	}
	if first.PassageCount != 30 || first.SourceCount != 30 || len(first.Rows[0]) != 29 || len(first.Rows[1]) != 30 {
		t.Errorf("sampled %d passages of %d sources in rows of %d and %d, want 30 of 30 in rows of 29 and 30",
			first.PassageCount, first.SourceCount, len(first.Rows[0]), len(first.Rows[1]))
	}
	for i := range first.Rows {
		if !slices.Equal(first.Rows[i], second.Rows[i]) {
			t.Errorf("two samples gave different row %d", i)
		}
	}
	// The first query leans on the first axis, so its similarity falls along the index.
	if !slices.IsSortedFunc(first.Rows[0], func(a, b float64) int {
		switch {
		case a > b:
			return -1
		case a < b:
			return 1
		}
		return 0
	}) {
		t.Errorf("the similarities do not follow the order of the passages: %v", first.Rows[0])
	}
}

func TestTheSearchRefusesAQueryOfAnotherWidth(t *testing.T) {
	search := newSearch()
	search.SetPassages(searchRecipe, passagesOf(3))
	_, err := search.SampleSimilarities(t.Context(), searchRecipe, []host.SimilarityQuery{{Vector: []float32{1, 0, 0}}})
	if !errors.Is(err, host.ErrInvalidQuery) {
		t.Errorf("got %v, want ErrInvalidQuery", err)
	}
}

func TestTheSearchRefusesARecipeOfAnotherModelAndAHostWithNoModel(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.SetPassages(searchRecipe, passagesOf(3))

	if _, err := search.SampleSimilarities(ctx, "other", nil); !errors.Is(err, host.ErrRecipeMismatch) {
		t.Errorf("sample gave %v", err)
	}
	if _, err := search.GetBestSimilarity(ctx, "other", []float32{1, 0}); !errors.Is(err, host.ErrRecipeMismatch) {
		t.Errorf("best similarity gave %v", err)
	}
	if _, err := search.GetSearchFloor(ctx, "other"); !errors.Is(err, host.ErrRecipeMismatch) {
		t.Errorf("floor read gave %v", err)
	}
	if err := search.SetSearchFloor(ctx, "other", 0.5); !errors.Is(err, host.ErrRecipeMismatch) {
		t.Errorf("floor set gave %v", err)
	}
	if _, found := search.StoredFloor("other"); found || search.FloorWrites() != 0 {
		t.Error("a refused floor was stored")
	}

	search.RemoveModel()
	if _, err := search.GetEmbeddingState(ctx); !errors.Is(err, host.ErrNoModel) {
		t.Errorf("state gave %v", err)
	}
	if _, err := search.EmbedTexts(ctx, []string{"a"}); !errors.Is(err, host.ErrNoModel) {
		t.Errorf("embedding gave %v", err)
	}
	if err := search.SetSearchFloor(ctx, searchRecipe, 0.5); !errors.Is(err, host.ErrNoModel) {
		t.Errorf("floor set gave %v", err)
	}
}

func TestTheSearchRefusesAFloorOutsideTheRange(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	for _, floor := range []float64{-0.1, 1.01, 3} {
		if err := search.SetSearchFloor(ctx, searchRecipe, floor); !errors.Is(err, host.ErrInvalidFloor) {
			t.Errorf("floor %v gave %v", floor, err)
		}
	}
	if _, found := search.StoredFloor(searchRecipe); found {
		t.Error("a floor outside the range was stored")
	}
	if err := search.SetSearchFloor(ctx, searchRecipe, 1); err != nil {
		t.Errorf("a floor of one gave %v", err)
	}
}

func TestSettingTheStoredFloorAgainChangesNothingAndZeroRemovesIt(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	for range 3 {
		if err := search.SetSearchFloor(ctx, searchRecipe, 0.55); err != nil {
			t.Fatal(err)
		}
	}
	got, err := search.GetSearchFloor(ctx, searchRecipe)
	want := host.SearchFloor{Floor: 0.55, Source: host.FloorFromStore, Stored: 0.55, Hint: 0.4, Default: 0.30}
	if err != nil || got != want {
		t.Fatalf("floor %+v (%v), want %+v", got, err, want)
	}

	if err := search.SetSearchFloor(ctx, searchRecipe, 0); err != nil {
		t.Fatal(err)
	}
	got, err = search.GetSearchFloor(ctx, searchRecipe)
	if err != nil || got.Floor != 0.4 || got.Source != host.FloorFromModelHint || got.Stored != 0 {
		t.Errorf("after removing the floor: %+v (%v), want the hint", got, err)
	}
	search.SetModel(hosttest.SearchModel{Name: "bge", Recipe: searchRecipe})
	got, err = search.GetSearchFloor(ctx, searchRecipe)
	if err != nil || got.Floor != 0.30 || got.Source != host.FloorFromDefault {
		t.Errorf("with no hint: %+v (%v), want the default", got, err)
	}
}

func TestTheEmbeddingStateIsReadAtEachCallSoALatePluginSeesIt(t *testing.T) {
	ctx := t.Context()
	search := newSearch()

	state, err := search.GetEmbeddingState(ctx)
	if err != nil || state.IsReady || state.Recipe != searchRecipe || state.Model != "bge" || state.FloorHint != 0.4 {
		t.Fatalf("before any chunk: %+v (%v)", state, err)
	}
	search.SetProgress(10, 4)
	state, err = search.GetEmbeddingState(ctx)
	if err != nil || state.IsReady || state.ChunkCount != 10 || state.EmbeddedCount != 4 {
		t.Errorf("while embedding: %+v (%v)", state, err)
	}
	search.SetProgress(10, 10)
	for range 3 {
		state, err = search.GetEmbeddingState(ctx)
		if err != nil || !state.IsReady {
			t.Errorf("once embedded, read again: %+v (%v)", state, err)
		}
	}
}

func TestTheSearchEmbedsWithinTheLimitsAndFailsWhenTheModelFails(t *testing.T) {
	ctx := t.Context()
	search := newSearch()

	got, err := search.EmbedTexts(ctx, []string{"a", "abc"})
	if err != nil || got.Recipe != searchRecipe || len(got.Vectors) != 2 || got.Vectors[1][0] != 3 {
		t.Fatalf("embedded %+v (%v)", got, err)
	}
	if _, err := search.EmbedTexts(ctx, make([]string, host.MaxEmbeddedTexts+1)); !errors.Is(err, host.ErrLimitExceeded) {
		t.Errorf("too many texts gave %v", err)
	}
	long := make([]byte, host.MaxEmbeddedBytes+1)
	if _, err := search.EmbedTexts(ctx, []string{string(long)}); !errors.Is(err, host.ErrLimitExceeded) {
		t.Errorf("a long text gave %v", err)
	}
	if batches := search.EmbeddedBatches(); len(batches) != 1 {
		t.Errorf("the model embedded %d batches, want only the one within the limits", len(batches))
	}

	search.FailEmbedding(errors.New("the plugin is not running"))
	if _, err := search.EmbedTexts(ctx, []string{"a"}); !errors.Is(err, host.ErrEmbeddingFailed) {
		t.Errorf("a failing model gave %v", err)
	}
}

func TestTheBestSimilarityIsTheNearestPassage(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.SetPassages(searchRecipe, []hosttest.SearchPassage{
		{Path: "a.md", Vector: []float32{0, 127}},
		{Path: "b.md", Vector: []float32{127, 0}},
	})

	best, err := search.GetBestSimilarity(ctx, searchRecipe, []float32{1, 0})
	if err != nil || best < 0.999 {
		t.Errorf("best %v (%v), want 1", best, err)
	}
	none, err := search.With("chemistry").GetBestSimilarity(ctx, searchRecipe, []float32{1, 0})
	if err != nil || none != 0 {
		t.Errorf("an empty vault gave %v (%v)", none, err)
	}
}

func TestTheSearchReportsTheIndexStatisticsOfTheVaultForTheModelInUse(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.SetProgress(12, 9)
	search.SetIndexStats(host.IndexStats{
		SourceCount: 3, Extensions: []host.ExtensionStats{{Extension: "md", SourceCount: 3, ChunkCount: 12}},
		DatabaseBytes: 4096, VectorBytes: 2048,
	})

	got, err := search.GetIndexStats(ctx)
	if err != nil || got.Recipe != searchRecipe || got.Model != "bge" || got.ChunkCount != 12 || got.VectorCount != 9 ||
		got.SourceCount != 3 || len(got.Extensions) != 1 || got.DatabaseBytes != 4096 || got.VectorBytes != 2048 {
		t.Errorf("statistics %+v (%v)", got, err)
	}
	other, err := search.With("chemistry").GetIndexStats(ctx)
	if err != nil || other.ChunkCount != 0 || other.SourceCount != 0 {
		t.Errorf("the other vault reported %+v (%v)", other, err)
	}
	if _, err := search.With("none").GetIndexStats(ctx); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("an unknown vault gave %v", err)
	}
	search.Deny(host.PermissionIndexRead)
	if _, err := search.GetIndexStats(ctx); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("a denied read gave %v", err)
	}
	search.RemoveModel()
	search = hosttest.NewSearch("physics")
	if _, err := search.GetIndexStats(ctx); !errors.Is(err, host.ErrNoModel) {
		t.Errorf("a host with no model gave %v", err)
	}
}
