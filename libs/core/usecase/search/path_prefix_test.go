package search_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// prefixSpy records the path prefix each of the three searches is asked.
type prefixSpy struct {
	lexical, named, nearest []string
}

func (s *prefixSpy) FindLexical(_ context.Context, _ domain.VaultID, _ string, _ []domain.SourceKind, _ int, _ bool, under string) ([]domain.Passage, error) {
	s.lexical = append(s.lexical, under)
	return nil, nil
}

func (s *prefixSpy) GetNamedPassages(_ context.Context, _ domain.VaultID, _ string, _ []domain.SourceKind, _ int, _ bool, under string) ([]domain.Passage, error) {
	s.named = append(s.named, under)
	return nil, nil
}

func (s *prefixSpy) FindNearest(_ context.Context, _ domain.VaultID, q sourceport.NearestQuery) ([]domain.Passage, error) {
	s.nearest = append(s.nearest, q.Under)
	return nil, nil
}

func TestASearchHandsThePathPrefixToEveryRanking(t *testing.T) {
	spy := &prefixSpy{}
	searching := search.New(spy, vaultreader.Readers{}, nil, oneWay{direction: newVector(+1)}, nil)

	p := search.Parameters{Limit: 5, PathPrefix: "library/"}
	if _, err := searching.Execute(t.Context(), domain.Vault{ID: "01VAULT"}, "heron", p); err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string][]string{"words": spy.lexical, "names": spy.named, "meaning": spy.nearest} {
		if len(got) != 1 || got[0] != "library/" {
			t.Errorf("the ranking by %s was asked for the prefix %q, want one call with library/", name, got)
		}
	}
}
