package service_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

// errInvalidArgument is how the client of the host reports a call the host rejected as invalid.
var errInvalidArgument = errors.New("invalid_argument")

// searchCalls is the queries the host's search received.
type searchCalls struct {
	mu      sync.Mutex
	queries []host.SearchResultQuery
}

func (s *searchCalls) getQueries() []host.SearchResultQuery {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.queries)
}

// resultStatements is the search the fake passes the queries on to.
type resultStatements interface {
	SearchText(ctx context.Context, query host.SearchResultQuery) ([]host.SearchResult, error)
}

// recordingSearch is the fake search that records every query and rejects a kind it does not know as
// the host does.
type recordingSearch struct {
	search resultStatements
	calls  *searchCalls
}

func (r recordingSearch) SearchText(ctx context.Context, query host.SearchResultQuery) ([]host.SearchResult, error) {
	r.calls.mu.Lock()
	r.calls.queries = append(r.calls.queries, query)
	r.calls.mu.Unlock()
	for _, kind := range query.Kinds {
		if kind != domain.KindNote && kind != domain.KindFile {
			return nil, fmt.Errorf("%w: %q is not a searchable source kind: use \"note\" or \"file\"", errInvalidArgument, kind)
		}
	}
	return r.search.SearchText(ctx, query)
}

func newRecordedSearch(t *testing.T, vault *hosttest.Vault, results []host.SearchResult) (*searchCalls, *tools.Registry) {
	t.Helper()
	search := hosttest.NewSearch(testVault)
	search.SetIndexedResults(results)
	texts := hosttest.NewTexts(testVault, readerID)
	calls := &searchCalls{}
	registry, err := service.New(vaultsOf(vault)).
		WithLibrary(func(id domain.VaultID) service.Library {
			on := texts.With(id)
			return service.Library{Versions: on, Texts: on, Search: recordingSearch{search: search.With(id), calls: calls}}
		}).
		Registry()
	if err != nil {
		t.Fatal(err)
	}
	return calls, registry
}

func TestLibrarySearchPassesUnknownKindToHostAndReturnsItsError(t *testing.T) {
	calls, registry := newRecordedSearch(t, hosttest.New(testVault), []host.SearchResult{
		{Path: "library/Tea.pdf", Text: "green tea leaves", Start: 0, Length: 16},
	})

	failure := callFailure(t, registry, testVault, "library_search", `{"query":"tea","kinds":["book"]}`, nil)
	want := `"book" is not a searchable source kind: use "note" or "file"`
	if !strings.Contains(failure, want) {
		t.Errorf("library_search of the kind book gave %q, want the host's error %q", failure, want)
	}
	queries := calls.getQueries()
	if len(queries) != 1 || !slices.Equal(queries[0].Kinds, []domain.SourceKind{"book"}) {
		t.Errorf("the host received %+v, want one query with the kind book", queries)
	}
}

func TestLibrarySearchPassesTheLimitToTheHostUnchanged(t *testing.T) {
	calls, registry := newRecordedSearch(t, hosttest.New(testVault), []host.SearchResult{
		{Path: "library/Tea.pdf", Text: "green tea leaves", Start: 0, Length: 16},
	})

	for _, arguments := range []string{`{"query":"tea","limit":500}`, `{"query":"tea"}`, `{"query":"tea","limit":0}`} {
		if failure := callFailure(t, registry, testVault, "library_search", arguments, nil); failure != "" {
			t.Fatalf("library_search %s rejected: %s", arguments, failure)
		}
	}
	queries := calls.getQueries()
	if len(queries) != 3 {
		t.Fatalf("the host received %d queries, want 3", len(queries))
	}
	for i, want := range []int{500, 0, 0} {
		if queries[i].Limit != want {
			t.Errorf("query %d was sent to the host with the limit %d, want %d", i, queries[i].Limit, want)
		}
	}
}
