package search_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

func TestSearchTextRequestsTwentyWhenLimitIsZeroOrLess(t *testing.T) {
	for _, limit := range []int{0, -1, -500} {
		finder := &recordingFinder{}

		if _, err := search.NewSearchText(finder).Execute(t.Context(), getMarshVault(), search.SearchResultQuery{Query: "glacier", Limit: limit}); err != nil {
			t.Fatal(err)
		}

		if len(finder.params) != 1 || finder.params[0].Limit != 20 {
			t.Errorf("a limit of %d requested %+v from the finder, want one search for 20 passages", limit, finder.params)
		}
	}
}

func TestSearchTextCapsALimitAbove500To500(t *testing.T) {
	if search.MaxSearchedResults != 500 {
		t.Errorf("MaxSearchedResults is %d, want 500", search.MaxSearchedResults)
	}
	for limit, want := range map[int]int{1: 1, 100: 100, 300: 300, 500: 500, 501: 500, 5000: 500} {
		finder := &recordingFinder{}

		if _, err := search.NewSearchText(finder).Execute(t.Context(), getMarshVault(), search.SearchResultQuery{Query: "glacier", Limit: limit}); err != nil {
			t.Fatal(err)
		}

		if len(finder.params) != 1 || finder.params[0].Limit != want {
			t.Errorf("a limit of %d requested %+v from the finder, want one search for %d passages", limit, finder.params, want)
		}
	}
}

func TestSearchTextReturns300ResultsWhenTheQueryAsksFor300(t *testing.T) {
	finder := &recordingFinder{}

	if _, err := search.NewSearchText(finder).Execute(t.Context(), getMarshVault(), search.SearchResultQuery{Query: "glacier", Limit: 300}); err != nil {
		t.Fatal(err)
	}

	p := finder.params[0]
	if p.Limit != 300 || p.Lexical != 300*5 || p.Sources != 300 || p.Named != 300 {
		t.Errorf("a limit of 300 requested %+v, want 300 results, 1500 word candidates, 300 by meaning and 300 by name", p)
	}
}

func TestSearchTextReturnsMoreThanThreeResultsOfOneFileWhenTheQueryAsksForThem(t *testing.T) {
	finder := &recordingFinder{}

	if _, err := search.NewSearchText(finder).Execute(t.Context(), getMarshVault(), search.SearchResultQuery{Query: "glacier", Limit: 50, ResultsPerSource: 10}); err != nil {
		t.Fatal(err)
	}

	if p := finder.params[0]; p.Each != 10 {
		t.Errorf("ten results per file requested %d of one file, want 10", p.Each)
	}
}

func TestSearchTextKeepsThreePerFileWhenTheQueryNamesNoNumber(t *testing.T) {
	for _, c := range []struct {
		limit, perSource, want int
	}{
		{limit: 20, perSource: 0, want: 3},
		{limit: 20, perSource: -4, want: 3},
		{limit: 5, perSource: 10, want: 5},
	} {
		finder := &recordingFinder{}

		query := search.SearchResultQuery{Query: "glacier", Limit: c.limit, ResultsPerSource: c.perSource}
		if _, err := search.NewSearchText(finder).Execute(t.Context(), getMarshVault(), query); err != nil {
			t.Fatal(err)
		}

		if got := finder.params[0].Each; got != c.want {
			t.Errorf("a limit of %d and %d per file requested %d of one file, want %d", c.limit, c.perSource, got, c.want)
		}
	}
}
