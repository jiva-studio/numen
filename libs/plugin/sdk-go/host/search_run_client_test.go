package host_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// runHost answers the search calls with fixed values and records the requests.
type runHost struct {
	numenv1connect.UnimplementedHostSearchServiceHandler
	code    v1.ErrorCode
	vault   string
	start   *v1.StartSearchRequest
	get     *v1.GetSearchRequest
	removed string
}

func (h *runHost) StartSearch(_ context.Context, req *connect.Request[v1.StartSearchRequest]) (*connect.Response[v1.StartSearchResponse], error) {
	h.vault, h.start = req.Header().Get("Numen-Vault"), req.Msg
	return connect.NewResponse(&v1.StartSearchResponse{Error: h.code, Id: "s1"}), nil
}

func (h *runHost) GetSearch(_ context.Context, req *connect.Request[v1.GetSearchRequest]) (*connect.Response[v1.GetSearchResponse], error) {
	h.get = req.Msg
	score := 0.5
	return connect.NewResponse(&v1.GetSearchResponse{Error: h.code, Search: &v1.SearchState{
		Id: "s1", IsRunning: true, IsPartial: true, IsStale: true, Progress: &v1.SearchProgress{Step: "reranking"},
		Results: []*v1.SearchResult{
			{Path: "books/Tides.pdf", Text: "the tide", ByteRange: &v1.Range{Start: 5, End: 15}, Kind: "pdf", Title: "Tides", Score: &score},
			{Path: "notes/Sea.md", Text: "# Sea", Kind: "note", Title: "Sea", IsWholeFile: true},
		},
		Errors:   []*v1.SearchProblem{{Line: 1, Column: 3, Message: "unknown function", Suggestion: "meaning"}},
		Warnings: []string{"w"}, NextPageToken: "next", TotalResults: 45,
		Reranks:   []*v1.SearchRerank{{ConnectionName: "Local", SentCount: 3, ReusedCount: 1}},
		ScoreKind: "rerank", Floor: 0.3, IndexGeneration: "e.2",
	}}), nil
}

func (h *runHost) DeleteSearch(_ context.Context, req *connect.Request[v1.DeleteSearchRequest]) (*connect.Response[v1.DeleteSearchResponse], error) {
	h.removed = req.Msg.GetId()
	return connect.NewResponse(&v1.DeleteSearchResponse{Error: h.code}), nil
}

func (h *runHost) ListSearchFunctions(context.Context, *connect.Request[v1.ListSearchFunctionsRequest]) (*connect.Response[v1.ListSearchFunctionsResponse], error) {
	return connect.NewResponse(&v1.ListSearchFunctionsResponse{Error: h.code, Revision: "r1", Functions: []*v1.SearchFunction{{
		Name: "meaning", Kind: "function", Level: "chunk", Signatures: []string{"meaning(string) -> SearchResults"},
		Summary: "passages close in meaning", Description: "Finds passages close in meaning.", Example: `meaning("tide")`,
	}}}), nil
}

func dialRuns(t *testing.T, h *runHost) *host.SearchClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostSearchServiceHandler(h))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return host.NewSearchClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestASearchClientStartsReadsAndRemovesASearch(t *testing.T) {
	h := &runHost{}
	c := dialRuns(t, h).With("v2")
	ctx := t.Context()

	id, err := c.StartSearch(ctx, host.SearchStart{Query: `words("tide")`, Priority: host.SearchPriorityInteractive})
	if err != nil || id != "s1" || h.vault != "v2" || h.start.GetQuery() != `words("tide")` || h.start.GetPriority() != v1.Priority_PRIORITY_INTERACTIVE || h.start.GetIsBackground() {
		t.Fatalf("start %q (%v), host received %v for %q", id, err, h.start, h.vault)
	}
	state, err := c.GetSearch(ctx, id, host.SearchRead{Wait: 2 * time.Second, PageSize: 20, PageToken: "t"})
	if err != nil || h.get.GetWaitMs() != 2000 || h.get.GetPageSize() != 20 || h.get.GetPageToken() != "t" {
		t.Fatalf("get %v, host received %v", err, h.get)
	}
	if !state.IsRunning || !state.IsPartial || !state.IsStale || state.Step != "reranking" || state.NextPageToken != "next" ||
		state.TotalResults != 45 || state.ScoreKind != "rerank" || state.Floor != 0.3 || state.IndexGeneration != "e.2" ||
		!slices.Equal(state.Warnings, []string{"w"}) ||
		!slices.Equal(state.Errors, []host.SearchProblem{{Line: 1, Column: 3, Message: "unknown function", Suggestion: "meaning"}}) ||
		!slices.Equal(state.Reranks, []host.SearchRerank{{ConnectionName: "Local", SentCount: 3, ReusedCount: 1}}) {
		t.Fatalf("state %+v", state)
	}
	passage, whole := state.Results[0], state.Results[1]
	if passage.Start != 5 || passage.Length != 10 || passage.Kind != "pdf" || passage.Title != "Tides" || *passage.Score != 0.5 || passage.IsWholeFile {
		t.Errorf("passage %+v", passage)
	}
	if !whole.IsWholeFile || whole.Score != nil || whole.Title != "Sea" {
		t.Errorf("whole file %+v", whole)
	}
	if err := c.DeleteSearch(ctx, id); err != nil || h.removed != "s1" {
		t.Errorf("remove %v, host removed %q", err, h.removed)
	}
	functions, err := c.ListSearchFunctions(ctx)
	if err != nil || functions.Revision != "r1" || len(functions.Functions) != 1 || functions.Functions[0].Example != `meaning("tide")` ||
		functions.Functions[0].Summary != "passages close in meaning" ||
		!slices.Equal(functions.Functions[0].Signatures, []string{"meaning(string) -> SearchResults"}) {
		t.Errorf("functions %+v (%v)", functions, err)
	}
}

func TestASearchClientMapsTheErrorsOfSearches(t *testing.T) {
	for code, want := range map[v1.ErrorCode]error{
		v1.ErrorCode_ERROR_CODE_NOT_FOUND: host.ErrSearchNotFound,
		v1.ErrorCode_ERROR_CODE_INVALID:   host.ErrInvalidPageToken,
	} {
		c := dialRuns(t, &runHost{code: code})
		if _, err := c.GetSearch(t.Context(), "s1", host.SearchRead{}); !errors.Is(err, want) {
			t.Errorf("%s: %v, want %v", code, err, want)
		}
	}
	c := dialRuns(t, &runHost{code: v1.ErrorCode_ERROR_CODE_LIMIT_EXCEEDED})
	if _, err := c.StartSearch(t.Context(), host.SearchStart{Query: "a"}); !errors.Is(err, host.ErrTooManySearches) {
		t.Errorf("limit exceeded: %v, want %v", err, host.ErrTooManySearches)
	}
}
