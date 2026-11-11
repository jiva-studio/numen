package search_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// prefixSpy records the path prefix each of the words and names searches is requested.
type prefixSpy struct {
	mu             sync.Mutex
	lexical, named []string
}

func (s *prefixSpy) FindLexical(_ context.Context, _ domain.VaultID, q source.LexicalQuery) ([]domain.Passage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lexical = append(s.lexical, q.PathPrefix)
	return nil, nil
}

func (s *prefixSpy) GetNamedPassages(_ context.Context, _ domain.VaultID, q source.LexicalQuery) ([]domain.Passage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.named = append(s.named, q.PathPrefix)
	return nil, nil
}

func TestSearchPassesPathPrefixToEveryRanking(t *testing.T) {
	spy := &prefixSpy{}
	nearest := &nearestFake{}
	searching := search.New(search.Index{Passages: spy, Nearest: nearest}, vaultreader.Readers{}, nil, oneWay{direction: newVector(+1)}, vectorscase.NewEmbedQueries(oneWay{direction: newVector(+1)}, nil), nil)

	p := search.Parameters{Limit: 5, PathPrefix: "library/"}
	if _, err := searching.Execute(t.Context(), domain.Vault{ID: "01VAULT"}, "heron", p); err != nil {
		t.Fatal(err)
	}
	meaning := make([]string, 0, len(nearest.coarse))
	for _, q := range nearest.coarse {
		meaning = append(meaning, q.PathPrefix)
	}
	for name, got := range map[string][]string{"words": spy.lexical, "names": spy.named, "meaning": meaning} {
		if len(got) != 1 || got[0] != "library/" {
			t.Errorf("the ranking by %s received a request for the prefix %q, want one call with library/", name, got)
		}
	}
}
