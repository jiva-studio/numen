package hostsearch

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

func TestSearchTextFusesTheParticipantsWithTheLimitAndTheResultsPerSource(t *testing.T) {
	finder := &askedFinder{}
	s := Search{TextSearch: search.NewSearchText(finder)}

	_, err := s.SearchText(t.Context(), getVault(), plugin.SearchResultQuery{Query: "heron reed", Limit: 300, ResultsPerSource: 9})
	if err != nil {
		t.Fatal(err)
	}
	p := finder.params
	if finder.query != "heron reed" || p.Limit != 300 || p.Each != 9 || p.Lexical == 0 || p.Sources == 0 || p.Named == 0 {
		t.Errorf("the search received %q with %+v, want every participant, 300 results and 9 per file", finder.query, p)
	}
}
