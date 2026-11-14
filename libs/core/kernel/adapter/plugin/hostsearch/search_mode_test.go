package hostsearch

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

type failingFinder struct {
	err   error
	calls int
}

func (f *failingFinder) Execute(context.Context, domain.Vault, string, search.Parameters) ([]domain.SearchResult, error) {
	f.calls++
	return nil, f.err
}

func TestSearchTextPassesTheModeTheRawWordsAndTheResultsPerSource(t *testing.T) {
	finder := &askedFinder{}
	s := Search{TextSearch: search.NewSearchText(finder)}

	_, err := s.SearchText(t.Context(), getVault(), plugin.SearchResultQuery{
		Query: "NEAR(heron reed, 3)", Limit: 300, ResultsPerSource: 9, Mode: plugin.SearchModeWords, IsRawWords: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	p := finder.params
	if finder.query != "NEAR(heron reed, 3)" || p.Limit != 300 || p.Each != 9 || !p.IsRawWords || p.Lexical == 0 || p.Sources != 0 || p.Named != 0 {
		t.Errorf("the search received %q with %+v, want the raw words alone, 300 results and 9 per file", finder.query, p)
	}
}

func TestSearchTextAsksOnlyTheMeaningInMeaningMode(t *testing.T) {
	finder := &askedFinder{}
	s := Search{TextSearch: search.NewSearchText(finder), Source: &stubEmbedder{model: getModel()}}

	if _, err := s.SearchText(t.Context(), getVault(), plugin.SearchResultQuery{Query: "heron", Mode: plugin.SearchModeMeaning}); err != nil {
		t.Fatal(err)
	}
	if p := finder.params; p.Sources == 0 || p.Lexical != 0 || p.Named != 0 {
		t.Errorf("meaning mode asked %+v, want the search source alone", p)
	}
}

func TestSearchTextRefusesMeaningModeWithoutAModel(t *testing.T) {
	finder := &failingFinder{}
	s := Search{TextSearch: search.NewSearchText(finder)}

	_, err := s.SearchText(t.Context(), getVault(), plugin.SearchResultQuery{Query: "heron", Mode: plugin.SearchModeMeaning})
	if !errors.Is(err, plugin.ErrNoModel) {
		t.Errorf("meaning mode with no model returned %v, want ErrNoModel", err)
	}
	if finder.calls != 0 {
		t.Errorf("the search ran %d times with no model", finder.calls)
	}
}

func TestSearchTextReturnsErrInvalidWordsForARawQueryTheIndexRefuses(t *testing.T) {
	finder := &failingFinder{err: sourcedomain.ErrInvalidWords}
	s := Search{TextSearch: search.NewSearchText(finder)}

	_, err := s.SearchText(t.Context(), getVault(), plugin.SearchResultQuery{Query: `"heron`, Mode: plugin.SearchModeWords, IsRawWords: true})
	if !errors.Is(err, plugin.ErrInvalidWords) {
		t.Errorf("a query the index refuses returned %v, want plugin.ErrInvalidWords", err)
	}
}
