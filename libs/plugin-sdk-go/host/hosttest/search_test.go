package hosttest_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

const searchRecipe = "local|bge|1024|512|mean|int8@0.4"

// newSearch is the search of a host that shows the vault "physics", knows "chemistry", and runs the model.
func newSearch() *hosttest.Search {
	search := hosttest.NewSearch("physics", "chemistry")
	search.SetModel(hosttest.SearchModel{Name: "bge", Recipe: searchRecipe, FloorHint: 0.4})
	search.SetEmbedder(func(text string) []float32 { return []float32{float32(len(text)), 1} })
	return search
}

func titlesOf(n int) []host.NoteTitle {
	out := make([]host.NoteTitle, n)
	for i := range out {
		out[i] = host.NoteTitle{Source: int64(i + 1), Title: fmt.Sprintf("title %d", i)}
	}
	return out
}

func vectorsOf(n int) []host.ChunkVector {
	out := make([]host.ChunkVector, n)
	for i := range out {
		out[i] = host.ChunkVector{Source: int64(i + 1), Vector: []byte{byte(i), 1}}
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
	if _, err := search.ListNoteTitles(ctx, 1); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("titles gave %v", err)
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
	search.SetNoteTitles(titlesOf(3))

	unknown := search.With("biology")
	if _, err := unknown.GetEmbeddingState(ctx); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("state gave %v", err)
	}
	if _, err := unknown.ListNoteTitles(ctx, 1); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("titles gave %v", err)
	}
	if err := unknown.SetSearchFloor(ctx, searchRecipe, 0.5); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("floor set gave %v", err)
	}

	titles, err := search.With("chemistry").ListNoteTitles(ctx, 10)
	if err != nil || len(titles) != 0 {
		t.Errorf("another vault listed %v (%v), want none", titles, err)
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

func TestTheSearchCapsEveryList(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.SetNoteTitles(titlesOf(host.MaxListedTitles + 50))
	search.SetChunkVectors(searchRecipe, vectorsOf(host.MaxListedVectors+50))

	for _, limit := range []int{0, -1, 1_000_000} {
		titles, err := search.ListNoteTitles(ctx, limit)
		if err != nil || len(titles) != host.MaxListedTitles {
			t.Errorf("titles with limit %d: %d (%v), want %d", limit, len(titles), err, host.MaxListedTitles)
		}
		vectors, err := search.ListSmallChunkVectors(ctx, searchRecipe, limit)
		if err != nil || len(vectors) != host.MaxListedVectors {
			t.Errorf("vectors with limit %d: %d (%v), want %d", limit, len(vectors), err, host.MaxListedVectors)
		}
	}
	titles, err := search.ListNoteTitles(ctx, 7)
	if err != nil || len(titles) != 7 {
		t.Errorf("a limit of 7 listed %d (%v)", len(titles), err)
	}
}

func TestTheSearchListsInTheOrderOfTheIndexEveryTime(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.SetNoteTitles(titlesOf(30))
	search.SetChunkVectors(searchRecipe, vectorsOf(30))

	first, err := search.ListNoteTitles(ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	second, err := search.ListNoteTitles(ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(first, second) || first[0].Source != 1 || first[19].Source != 20 {
		t.Errorf("two reads gave %v and %v", first, second)
	}
	prefix, err := search.ListNoteTitles(ctx, 5)
	if err != nil || !slices.Equal(prefix, first[:5]) {
		t.Errorf("a shorter read gave %v (%v), want the head of the longer one", prefix, err)
	}
	vectors, err := search.ListSmallChunkVectors(ctx, searchRecipe, 30)
	if err != nil {
		t.Fatal(err)
	}
	for i, one := range vectors {
		if one.Source != int64(i+1) {
			t.Fatalf("vector %d belongs to source %d", i, one.Source)
		}
	}
}

func TestTheSearchRefusesARecipeOfAnotherModelAndAHostWithNoModel(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.SetChunkVectors(searchRecipe, vectorsOf(3))

	if _, err := search.ListSmallChunkVectors(ctx, "other", 10); !errors.Is(err, host.ErrRecipeMismatch) {
		t.Errorf("vectors gave %v", err)
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
	if state, _ = search.GetEmbeddingState(ctx); state.IsReady || state.ChunkCount != 10 || state.EmbeddedCount != 4 {
		t.Errorf("while embedding: %+v", state)
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

func TestTheBestSimilarityIsTheNearestStoredVector(t *testing.T) {
	ctx := t.Context()
	search := newSearch()
	search.SetChunkVectors(searchRecipe, []host.ChunkVector{
		{Source: 1, Vector: []byte{0, 127}},
		{Source: 2, Vector: []byte{127, 0}},
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
