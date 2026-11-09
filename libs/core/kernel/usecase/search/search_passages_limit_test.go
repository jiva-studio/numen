package search_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

func TestSearchPassagesRequestsTwentyWhenLimitIsZeroOrLess(t *testing.T) {
	for _, limit := range []int{0, -1, -500} {
		finder := &recordingFinder{}

		if _, err := search.NewSearchPassages(finder).Execute(t.Context(), getMarshVault(), search.PassageQuery{Query: "glacier", Limit: limit}); err != nil {
			t.Fatal(err)
		}

		if len(finder.params) != 1 || finder.params[0].Limit != 20 {
			t.Errorf("a limit of %d requested %+v from the finder, want one search for 20 passages", limit, finder.params)
		}
	}
}

func TestSearchPassagesRequestsAtMostOneHundredFromTheFinder(t *testing.T) {
	for limit, want := range map[int]int{1: 1, 100: 100, 101: 100, 500: 100} {
		finder := &recordingFinder{}

		if _, err := search.NewSearchPassages(finder).Execute(t.Context(), getMarshVault(), search.PassageQuery{Query: "glacier", Limit: limit}); err != nil {
			t.Fatal(err)
		}

		if len(finder.params) != 1 || finder.params[0].Limit != want {
			t.Errorf("a limit of %d requested %+v from the finder, want one search for %d passages", limit, finder.params, want)
		}
	}
}
