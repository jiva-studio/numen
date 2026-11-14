package hosttest_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

func TestSearchTextTruncatesALongTextToTheExcerptAroundTheHit(t *testing.T) {
	search := newSearch()
	long := strings.Repeat("filler ", 400) + "heron " + strings.Repeat("filler ", 400)
	search.SetIndexedResults([]host.SearchResult{
		{Path: "notes/Long.md", Text: long, Start: 0, Length: len(long)},
		{Path: "notes/Short.md", Text: "a heron", Start: 0, Length: 7},
	})

	got, err := search.SearchText(t.Context(), host.SearchResultQuery{Query: "heron"})
	if err != nil || len(got) != 2 {
		t.Fatalf("found %d, %v, want both results", len(got), err)
	}
	if text := got[0].Text; len(text) > host.MaxExcerptBytes || !strings.Contains(text, "heron") {
		t.Errorf("the long result has %d bytes, heron %v, want at most %d bytes around the hit", len(text), strings.Contains(text, "heron"), host.MaxExcerptBytes)
	}
	if got[0].Length != len(long) {
		t.Errorf("the long result has the length %d, want the whole chunk, %d", got[0].Length, len(long))
	}
	if got[1].Text != "a heron" {
		t.Errorf("the short result has the text %q, want it whole", got[1].Text)
	}
}

func TestSearchTextKeepsTheScoresOfTheIndexedResults(t *testing.T) {
	search := newSearch()
	similarity, wordScore := 0.62, 3.5
	search.SetIndexedResults([]host.SearchResult{
		{ChunkID: "c1", Path: "notes/Heron.md", Text: "a heron", Similarity: &similarity, WordScore: &wordScore},
		{ChunkID: "c2", Path: "notes/Reed.md", Text: "a heron in the reed"},
	})

	got, err := search.SearchText(t.Context(), host.SearchResultQuery{Query: "heron"})
	if err != nil || len(got) != 2 {
		t.Fatalf("found %d, %v, want both results", len(got), err)
	}
	if got[0].Similarity == nil || *got[0].Similarity != 0.62 || got[0].WordScore == nil || *got[0].WordScore != 3.5 {
		t.Errorf("the scored result has the similarity %v and the word score %v, want 0.62 and 3.5", got[0].Similarity, got[0].WordScore)
	}
	if got[1].Similarity != nil || got[1].WordScore != nil {
		t.Errorf("the result with no scores has the similarity %v and the word score %v, want none", got[1].Similarity, got[1].WordScore)
	}
}

func TestScoreChunksReturnsTheFloorHintOfTheModelWhenNoFloorIsStored(t *testing.T) {
	search := newSearch()
	search.SetIndexedResults([]host.SearchResult{{ChunkID: "c1", Path: "notes/Heron.md", Text: "a heron"}})

	got, err := search.ScoreChunks(t.Context(), host.ScoreQuery{ChunkIDs: []string{"c1"}, Conditions: []host.ScoreCondition{{Meaning: "herons"}}})
	if err != nil || got.Floor != 0.4 {
		t.Errorf("returned the floor %v, %v, want the hint of the model, 0.4", got.Floor, err)
	}
}
