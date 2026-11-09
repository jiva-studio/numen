package search_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// prefixSpy records the path prefix each of the three searches is requested.
type prefixSpy struct {
	lexical, named, nearest []string
}

func (s *prefixSpy) FindLexical(_ context.Context, _ domain.VaultID, q sourceport.LexicalQuery) ([]domain.Passage, error) {
	s.lexical = append(s.lexical, q.PathPrefix)
	return nil, nil
}

func (s *prefixSpy) GetNamedPassages(_ context.Context, _ domain.VaultID, q sourceport.LexicalQuery) ([]domain.Passage, error) {
	s.named = append(s.named, q.PathPrefix)
	return nil, nil
}

func (s *prefixSpy) FindNearest(_ context.Context, _ domain.VaultID, q sourceport.NearestQuery) ([]domain.Passage, error) {
	s.nearest = append(s.nearest, q.PathPrefix)
	return nil, nil
}

func TestSearchPassesPathPrefixToEveryRanking(t *testing.T) {
	spy := &prefixSpy{}
	searching := search.New(spy, vaultreader.Readers{}, nil, oneWay{direction: newVector(+1)}, nil)

	p := search.Parameters{Limit: 5, PathPrefix: "library/"}
	if _, err := searching.Execute(t.Context(), domain.Vault{ID: "01VAULT"}, "heron", p); err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string][]string{"words": spy.lexical, "names": spy.named, "meaning": spy.nearest} {
		if len(got) != 1 || got[0] != "library/" {
			t.Errorf("the ranking by %s received a request for the prefix %q, want one call with library/", name, got)
		}
	}
}
