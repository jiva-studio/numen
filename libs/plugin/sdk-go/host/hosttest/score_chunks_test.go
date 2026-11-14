package hosttest_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

const (
	heronText = "The gray heron waits in the reeds"
	frogText  = "Frogs sing at night"
)

func newScoredSearch() *hosttest.Search {
	search := newSearch()
	search.SetIndexedResults([]host.SearchResult{
		{ChunkID: "c1", Path: "library/Birds.pdf", Text: heronText, Start: 400, Length: 700},
		{ChunkID: "c2", Path: "notes/Frogs.md", Text: frogText, Start: 0, Length: 19},
	})
	vectors := map[string][]float32{"herons": {1, 0}, heronText: {1, 1}, frogText: {0, 1}}
	search.SetEmbedder(func(text string) []float32 { return vectors[text] })
	return search
}

func TestTheFakeScoresAWordsConditionByEveryWordOfTheQuery(t *testing.T) {
	got, err := newScoredSearch().ScoreChunks(t.Context(), host.ScoreQuery{
		ChunkIDs: []string{"c1", "c2"}, Conditions: []host.ScoreCondition{{Words: "heron reeds"}},
	})
	if err != nil || len(got.Chunks) != 2 {
		t.Fatalf("scored %+v, %v, want both chunks", got, err)
	}
	if one := got.Chunks[0]; one.ChunkID != "c1" || !one.Conditions[0].IsMatch || one.Conditions[0].Similarity != nil {
		t.Errorf("the heron chunk scored %+v, want a match with no similarity", one)
	}
	if one := got.Chunks[1]; one.ChunkID != "c2" || one.Conditions[0].IsMatch {
		t.Errorf("the frog chunk scored %+v, want no match", one)
	}
	if got.Floor != 0 {
		t.Errorf("a call with no meaning condition returned the floor %v, want zero", got.Floor)
	}
}

func TestTheFakeScoresAMeaningConditionAgainstTheFloorOfTheVault(t *testing.T) {
	search := newScoredSearch()
	query := host.ScoreQuery{ChunkIDs: []string{"c1", "c2"}, Conditions: []host.ScoreCondition{{Meaning: "herons"}}}

	got, err := search.ScoreChunks(t.Context(), query)
	if err != nil || len(got.Chunks) != 2 {
		t.Fatalf("scored %+v, %v, want both chunks", got, err)
	}
	if got.Floor != 0.4 {
		t.Errorf("the floor is %v, want the hint of the model, 0.4", got.Floor)
	}
	heron, frog := got.Chunks[0].Conditions[0], got.Chunks[1].Conditions[0]
	if heron.Similarity == nil || *heron.Similarity < 0.70 || *heron.Similarity > 0.71 || !heron.IsMatch {
		t.Errorf("the heron chunk scored %+v, want about 0.707 and a match", heron)
	}
	if frog.Similarity == nil || *frog.Similarity != 0 || frog.IsMatch {
		t.Errorf("the frog chunk scored %+v, want the similarity 0 and no match", frog)
	}

	if err := search.SetSearchFloor(t.Context(), searchRecipe, 0.8); err != nil {
		t.Fatal(err)
	}
	stored, err := search.ScoreChunks(t.Context(), query)
	if err != nil || stored.Floor != 0.8 || stored.Chunks[0].Conditions[0].IsMatch {
		t.Errorf("under a stored floor of 0.8 scored %+v, %v, want no match", stored, err)
	}
}

func TestTheFakeOmitsAnUnknownChunkIDAndARepeatedOne(t *testing.T) {
	got, err := newScoredSearch().ScoreChunks(t.Context(), host.ScoreQuery{
		ChunkIDs: []string{"missing", "c2", "c1", "c2"}, Conditions: []host.ScoreCondition{{Words: "frogs"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Chunks) != 2 || got.Chunks[0].ChunkID != "c2" || got.Chunks[1].ChunkID != "c1" {
		t.Errorf("scored %+v, want c2 and then c1, once each", got.Chunks)
	}
}

func TestTheFakeRefusesScoreChunksOverTheLimits(t *testing.T) {
	search := newScoredSearch()
	ids := make([]string, host.MaxScoredChunks+1)
	for i := range ids {
		ids[i] = "c" + strconv.Itoa(i+10)
	}
	conditions := make([]host.ScoreCondition, host.MaxScoredConditions+1)
	for i := range conditions {
		conditions[i] = host.ScoreCondition{Words: "heron"}
	}
	long := strings.Repeat("a", 8193)

	for name, query := range map[string]host.ScoreQuery{
		"501 chunks":    {ChunkIDs: ids, Conditions: conditions[:1]},
		"17 conditions": {ChunkIDs: []string{"c1"}, Conditions: conditions},
		"long words":    {ChunkIDs: []string{"c1"}, Conditions: []host.ScoreCondition{{Words: long}}},
		"long meaning":  {ChunkIDs: []string{"c1"}, Conditions: []host.ScoreCondition{{Meaning: long}}},
	} {
		if _, err := search.ScoreChunks(t.Context(), query); !errors.Is(err, host.ErrLimitExceeded) {
			t.Errorf("%s returned %v, want ErrLimitExceeded", name, err)
		}
	}
	if _, err := search.ScoreChunks(t.Context(), host.ScoreQuery{ChunkIDs: ids[:host.MaxScoredChunks], Conditions: conditions[:host.MaxScoredConditions]}); err != nil {
		t.Errorf("the most chunks and conditions returned %v, want an answer", err)
	}
}

func TestTheFakeRefusesAConditionWithNoTextOrBothTexts(t *testing.T) {
	for name, condition := range map[string]host.ScoreCondition{
		"no text":    {},
		"both texts": {Meaning: "herons", Words: "heron"},
	} {
		_, err := newScoredSearch().ScoreChunks(t.Context(), host.ScoreQuery{ChunkIDs: []string{"c1"}, Conditions: []host.ScoreCondition{condition}})
		if !errors.Is(err, host.ErrInvalidCondition) {
			t.Errorf("a condition with %s returned %v, want ErrInvalidCondition", name, err)
		}
	}
}

func TestTheFakeAnswersWordsWithoutAModelAndRefusesMeaning(t *testing.T) {
	search := newScoredSearch()
	search.RemoveModel()

	got, err := search.ScoreChunks(t.Context(), host.ScoreQuery{ChunkIDs: []string{"c1"}, Conditions: []host.ScoreCondition{{Words: "heron"}}})
	if err != nil || len(got.Chunks) != 1 || !got.Chunks[0].Conditions[0].IsMatch {
		t.Errorf("words with no model scored %+v, %v, want a match", got, err)
	}
	if _, err := search.ScoreChunks(t.Context(), host.ScoreQuery{ChunkIDs: []string{"c1"}, Conditions: []host.ScoreCondition{{Meaning: "herons"}}}); !errors.Is(err, host.ErrNoModel) {
		t.Errorf("meaning with no model returned %v, want ErrNoModel", err)
	}
}

func TestTheFakeRefusesScoreChunksWithoutThePermission(t *testing.T) {
	search := newScoredSearch()
	search.Deny(host.PermissionIndexRead)

	if _, err := search.ScoreChunks(t.Context(), host.ScoreQuery{ChunkIDs: []string{"c1"}, Conditions: []host.ScoreCondition{{Words: "heron"}}}); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("without the permission it returned %v", err)
	}
}

func TestTheFakeSearchTextReturnsMoreThanThreeOfOneFileWhenAsked(t *testing.T) {
	search := newSearch()
	var many []host.SearchResult
	for i := range 6 {
		many = append(many, host.SearchResult{Path: "library/Birds.pdf", Text: "heron", Start: i * 10})
	}
	search.SetIndexedResults(many)

	got, err := search.SearchText(t.Context(), host.SearchResultQuery{Query: "heron", ResultsPerSource: 5})
	if err != nil || len(got) != 5 {
		t.Errorf("five per file found %d, %v, want 5", len(got), err)
	}
}

func TestTheFakeSearchTextRefusesRawWordsOutsideWordsMode(t *testing.T) {
	search := newSearch()
	search.SetIndexedResults(getIndexedResults())

	for _, mode := range []host.SearchMode{host.SearchModeHybrid, host.SearchModeMeaning} {
		if _, err := search.SearchText(t.Context(), host.SearchResultQuery{Query: "heron", Mode: mode, IsRawWords: true}); err == nil {
			t.Errorf("raw words in mode %v returned no error", mode)
		}
	}
	got, err := search.SearchText(t.Context(), host.SearchResultQuery{Query: "heron", Mode: host.SearchModeWords, IsRawWords: true})
	if err != nil || len(got) != 3 {
		t.Errorf("raw words in words mode found %d, %v, want the three results with the word", len(got), err)
	}
}
