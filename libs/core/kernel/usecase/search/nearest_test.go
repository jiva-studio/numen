package search_test

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const nearestVault domain.VaultID = "01NEAREST"

func newNearestQuery(limit int) source.NearestQuery {
	return source.NearestQuery{Recipe: "model", Vector: newVector(+1), Limit: limit, Floor: -1}
}

func holdsEnclosing(f *nearestFake, names ...domain.ChunkID) {
	f.enclosing = map[domain.ChunkID]domain.SearchResult{}
	for _, n := range names {
		f.enclosing[n] = domain.SearchResult{ChunkID: n, Path: string(n)}
	}
}

func TestANearestSearchAnswersTheMostSimilarFirst(t *testing.T) {
	f := &nearestFake{}
	f.holdsSimilarity("middle", 0.5)
	f.holdsSimilarity("far", -0.4)
	f.holdsSimilarity("near", 0.9)
	holdsEnclosing(f, "middle", "far", "near")

	got, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, newNearestQuery(10))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"near", "middle", "far"}; !slices.Equal(chunksOf(got), want) {
		t.Errorf("got %v, want %v", chunksOf(got), want)
	}
}

func TestANearestSearchKeepsTheCoarseOrderAmongEqualSimilarities(t *testing.T) {
	f := &nearestFake{}
	var want []string
	var ids []domain.ChunkID
	for i := range 6 {
		name := fmt.Sprintf("twin-%d", i)
		f.holdsSimilarity(domain.ChunkID(name), 0.5)
		ids = append(ids, domain.ChunkID(name))
		want = append(want, name)
	}
	holdsEnclosing(f, ids...)

	got, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, newNearestQuery(10))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(chunksOf(got), want) {
		t.Errorf("got %v, want the order of the coarse pass %v", chunksOf(got), want)
	}
}

func TestANearestSearchKeepsAChunkAtTheFloorAndDropsOneBelowIt(t *testing.T) {
	f := &nearestFake{}
	f.holdsSimilarity("held", 0.5)
	holdsEnclosing(f, "held")
	exact := vector.SimilarityBytes(newVector(+1), f.vectors["held"])
	q := newNearestQuery(10)

	q.Floor = exact
	got, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, q)
	if err != nil || len(got) != 1 {
		t.Errorf("at the floor got %v, %v, want the chunk", chunksOf(got), err)
	}
	q.Floor = math.Nextafter(exact, 2)
	got, err = search.NewFindNearest(f).Execute(t.Context(), nearestVault, q)
	if err != nil || len(got) != 0 {
		t.Errorf("above the floor got %v, %v, want nothing", chunksOf(got), err)
	}
}

func TestANearestSearchAsksTheCoarsePassForSeveralTimesTheLimit(t *testing.T) {
	for _, limit := range []int{1, 3} {
		f := &nearestFake{}
		q := newNearestQuery(limit)
		q.Recipe, q.PathPrefix = "recipe-x", "library/"
		if _, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, q); err != nil {
			t.Fatal(err)
		}
		if len(f.coarse) != 1 || f.coarse[0].Limit != limit*8 || f.coarse[0].Recipe != "recipe-x" || f.coarse[0].PathPrefix != "library/" {
			t.Errorf("limit %d asked the coarse pass %+v, want %d candidates under the recipe and the prefix", limit, f.coarse, limit*8)
		}
	}
}

func TestANearestSearchReturnsAtMostTheLimitAndPassesTheKinds(t *testing.T) {
	f := &nearestFake{}
	for i := range 5 {
		f.holdsSimilarity(domain.ChunkID(fmt.Sprintf("c%d", i)), 0.9-float64(i)/10)
	}
	holdsEnclosing(f, "c0", "c1", "c2", "c3", "c4")
	q := newNearestQuery(2)
	q.Kinds = []domain.SourceKind{domain.KindNote}

	got, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, q)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"c0", "c1"}; !slices.Equal(chunksOf(got), want) {
		t.Errorf("got %v, want %v", chunksOf(got), want)
	}
	if len(f.stored) != 1 || !slices.Equal(f.stored[0].Kinds, q.Kinds) || f.stored[0].Recipe != "model" {
		t.Errorf("the vectors were asked for as %+v, want the kinds and the recipe of the query", f.stored)
	}
}

func TestANearestSearchHasNothingToSayWithoutAVectorAndRefusesNoLimit(t *testing.T) {
	f := &nearestFake{}
	f.holdsSimilarity("held", 0.5)
	q := newNearestQuery(5)
	q.Vector = nil
	if got, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, q); err != nil || got != nil || len(f.coarse) != 0 {
		t.Errorf("an empty vector got %v, %v after %d index calls", got, err, len(f.coarse))
	}
	for _, limit := range []int{0, -2} {
		if got, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, newNearestQuery(limit)); err == nil || got != nil {
			t.Errorf("limit %d got %v, %v", limit, got, err)
		}
	}
}

func TestANearestSearchAnswersNothingWhenTheVectorsAreOfAnotherRecipe(t *testing.T) {
	f := &nearestFake{coarseErr: fmt.Errorf("%w: dimension mismatch", source.ErrOtherRecipe)}
	got, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, newNearestQuery(5))
	if err != nil || got != nil {
		t.Errorf("got %v, %v, want silence", got, err)
	}
}

func TestANearestSearchReportsWhatTheIndexCannotRead(t *testing.T) {
	broken := errors.New("disk")
	f := &nearestFake{coarseErr: broken}
	if _, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, newNearestQuery(5)); !errors.Is(err, broken) {
		t.Errorf("coarse: got %v", err)
	}
	f = &nearestFake{storedErr: broken}
	f.holdsSimilarity("held", 0.5)
	if _, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, newNearestQuery(5)); !errors.Is(err, broken) {
		t.Errorf("vectors: got %v", err)
	}
}

func TestANearestSearchRefusesAVectorOfAnotherWidth(t *testing.T) {
	f := &nearestFake{}
	f.holdsSimilarity("held", 0.5)
	q := newNearestQuery(5)
	q.Vector = newVector(+1)[:512]
	if _, err := search.NewFindNearest(f).Execute(t.Context(), nearestVault, q); err == nil {
		t.Error("a query of 512 dimensions was compared to vectors of 1024")
	}
}

func TestTheBestSimilarityIsThatOfTheNearestChunkWhateverItIs(t *testing.T) {
	f := &nearestFake{}
	f.holdsSimilarity("a", -0.8)
	f.holdsSimilarity("b", -0.3)
	got, err := search.NewGetBestSimilarity(f).Execute(t.Context(), nearestVault, "model", newVector(+1))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-(-0.3)) > 0.01 {
		t.Errorf("got %v, want -0.3: no floor applies", got)
	}
	if len(f.coarse) != 1 || f.coarse[0].Limit != 8 || f.coarse[0].PathPrefix != "" {
		t.Errorf("the coarse pass was asked %+v, want 8 candidates over the whole vault", f.coarse)
	}
}

func TestTheBestSimilarityIsZeroWhereThereIsNothingToCompare(t *testing.T) {
	f := &nearestFake{}
	if got, err := search.NewGetBestSimilarity(f).Execute(t.Context(), nearestVault, "model", newVector(+1)); err != nil || got != 0 {
		t.Errorf("an empty vault got %v, %v", got, err)
	}
	if got, err := search.NewGetBestSimilarity(f).Execute(t.Context(), nearestVault, "model", nil); err != nil || got != 0 {
		t.Errorf("no vector got %v, %v", got, err)
	}
	other := &nearestFake{coarseErr: source.ErrOtherRecipe}
	if got, err := search.NewGetBestSimilarity(other).Execute(t.Context(), nearestVault, "model", newVector(+1)); err != nil || got != 0 {
		t.Errorf("another recipe got %v, %v", got, err)
	}
}

func TestTheBestSimilarityReportsWhatTheIndexCannotRead(t *testing.T) {
	broken := errors.New("disk")
	f := &nearestFake{coarseErr: broken}
	if _, err := search.NewGetBestSimilarity(f).Execute(t.Context(), nearestVault, "model", newVector(+1)); !errors.Is(err, broken) {
		t.Errorf("coarse: got %v", err)
	}
	f = &nearestFake{storedErr: broken}
	f.holdsSimilarity("held", 0.5)
	if _, err := search.NewGetBestSimilarity(f).Execute(t.Context(), nearestVault, "model", newVector(+1)); !errors.Is(err, broken) {
		t.Errorf("vectors: got %v", err)
	}
}

func chunksOf(matches []searchsource.Match) []string {
	out := make([]string, 0, len(matches))
	for _, one := range matches {
		out = append(out, string(one.ChunkID))
	}
	return out
}
