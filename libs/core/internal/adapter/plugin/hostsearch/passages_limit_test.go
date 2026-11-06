package hostsearch

import (
	"testing"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

func TestSearchPassagesAsksTheSearchForTwentyWhenTheLimitIsZeroOrLess(t *testing.T) {
	for _, limit := range []int{0, -1, -500} {
		finder := &recordingFinder{}
		s := Search{Finds: finder}

		if _, err := s.SearchPassages(t.Context(), getVault(), pluginport.PluginPassageQuery{Query: "glacier", Limit: limit}); err != nil {
			t.Fatal(err)
		}

		if len(finder.params) != 1 || finder.params[0].Limit != 20 {
			t.Errorf("a limit of %d asked the search for %+v, want one search for 20 passages", limit, finder.params)
		}
	}
}

func TestSearchPassagesAsksTheSearchForAtMostOneHundred(t *testing.T) {
	for limit, want := range map[int]int{1: 1, 100: 100, 101: 100, 500: 100} {
		finder := &recordingFinder{}
		s := Search{Finds: finder}

		if _, err := s.SearchPassages(t.Context(), getVault(), pluginport.PluginPassageQuery{Query: "glacier", Limit: limit}); err != nil {
			t.Fatal(err)
		}

		if len(finder.params) != 1 || finder.params[0].Limit != want {
			t.Errorf("a limit of %d asked the search for %+v, want one search for %d passages", limit, finder.params, want)
		}
	}
}
