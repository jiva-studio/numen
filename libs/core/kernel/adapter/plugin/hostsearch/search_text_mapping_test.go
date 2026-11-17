package hostsearch

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

// askedFinder returns a search with fixed passages and records the query and the parameters.
type askedFinder struct {
	found  []domain.SearchResult
	query  string
	params search.Parameters
}

func (f *askedFinder) Execute(_ context.Context, _ domain.Vault, query string, p search.Parameters) ([]domain.SearchResult, error) {
	f.query, f.params = query, p
	return f.found, nil
}

func TestSearchTextPassesPluginQueryToTheTextSearch(t *testing.T) {
	finder := &askedFinder{found: []domain.SearchResult{{Path: "library/Birds.pdf", Text: "the gray heron"}}}
	s := Search{TextSearch: search.NewSearchText(finder)}

	got, err := s.SearchText(t.Context(), getVault(), plugin.SearchResultQuery{
		Query: "heron", Limit: 7, Kinds: []domain.SourceKind{domain.KindFile}, PathPrefix: "library/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != "library/Birds.pdf" {
		t.Errorf("returned %+v, want the passage the search found", got)
	}
	p := finder.params
	if finder.query != "heron" || p.Limit != 7 || p.PathPrefix != "library/" || len(p.Kinds) != 1 || p.Kinds[0] != domain.KindFile {
		t.Errorf("the search received %q with %+v, want heron, limit 7, the prefix and the kind", finder.query, p)
	}
}
