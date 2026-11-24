package searchtest_test

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
