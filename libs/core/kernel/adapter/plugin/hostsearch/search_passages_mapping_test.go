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
	found  []domain.Passage
	query  string
	params search.Parameters
}

func (f *askedFinder) Execute(_ context.Context, _ domain.Vault, query string, p search.Parameters) ([]domain.Passage, error) {
	f.query, f.params = query, p
	return f.found, nil
}

func TestSearchPassagesPassesPluginQueryToThePassageSearch(t *testing.T) {
	finder := &askedFinder{found: []domain.Passage{{Source: "library/Birds.pdf", Text: "the gray heron"}}}
	s := Search{Passages: search.NewSearchPassages(finder)}

	got, err := s.SearchPassages(t.Context(), getVault(), plugin.PassageQuery{
		Query: "heron", Limit: 7, Kinds: []domain.SourceKind{domain.KindFile}, PathPrefix: "library/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Source != "library/Birds.pdf" {
		t.Errorf("returned %+v, want the passage the search found", got)
	}
	p := finder.params
	if finder.query != "heron" || p.Limit != 7 || p.PathPrefix != "library/" || len(p.Kinds) != 1 || p.Kinds[0] != domain.KindFile {
		t.Errorf("the search received %q with %+v, want heron, limit 7, the prefix and the kind", finder.query, p)
	}
}
