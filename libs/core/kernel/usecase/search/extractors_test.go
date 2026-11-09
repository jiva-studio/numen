package search

import "testing"

func TestASearchExtractsAsManySourcesAsItIsSetTo(t *testing.T) {
	if got := (Search{}).WithExtractors(7).GetExtractors(); got != 7 {
		t.Errorf("a search extracts %d sources at once, want 7", got)
	}
}

func TestASearchWithNoLimitExtractsTheDefaultNumberOfSources(t *testing.T) {
	for _, set := range []int{0, -1} {
		if got := (Search{}).WithExtractors(set).GetExtractors(); got != defaultExtractors {
			t.Errorf("set to %d, a search extracts %d sources at once, want %d", set, got, defaultExtractors)
		}
	}
}
