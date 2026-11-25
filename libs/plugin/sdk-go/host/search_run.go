package host

import (
	"context"
	"time"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// SearchPriority orders the searches waiting to run.
//
//numen:api
type SearchPriority int

const (
	// SearchPriorityBackground is work no person waits for.
	SearchPriorityBackground SearchPriority = iota
	// SearchPriorityInteractive is a search a person or an agent waits for; it runs first.
	SearchPriorityInteractive
)

// SearchStart is a query of the search language to run. A background search shows no task.
//
//numen:api
type SearchStart struct {
	Query        string
	Priority     SearchPriority
	IsBackground bool
}

// SearchRead is how to read a search. A zero Wait answers at once and a zero PageSize is 20.
//
//numen:api
type SearchRead struct {
	Wait      time.Duration
	PageSize  int
	PageToken string
}

// SearchState is a search and one page of its results. IsPartial is set while a running search holds the
// results before its rerank stage, and IsStale once the index of the vault changed after the start.
//
//numen:api
type SearchState struct {
	ID              string
	IsRunning       bool
	IsPartial       bool
	IsStale         bool
	Step            string
	Results         []SearchResult
	Errors          []SearchProblem
	Warnings        []string
	NextPageToken   string
	TotalResults    int
	Reranks         []SearchRerank
	ScoreKind       string
	Floor           float64
	IndexGeneration string
}

// SearchProblem is an error of a query. Line and Column count from 1 and are zero without a position; EndLine
// and EndColumn are the last character of a problem that covers more than one, else zero.
//
//numen:api
type SearchProblem struct {
	Line, Column       int
	EndLine, EndColumn int
	Message            string
	Suggestion         string
}

// SearchRerank is the connection that answered one rerank stage.
//
//numen:api
type SearchRerank struct {
	ConnectionName string
	SentCount      int
	ReusedCount    int
}

// SearchFunction is a function, a variable or a stage of the search language.
//
//numen:api
type SearchFunction struct {
	Name        string
	Kind        string
	Level       string
	Signatures  []string
	Summary     string
	Description string
	Example     string
}

// SearchFunctions is the search language and its revision, which changes whenever the list does.
//
//numen:api
type SearchFunctions struct {
	Functions []SearchFunction
	Revision  string
}

func writeSearchPriority(p SearchPriority) v1.Priority {
	if p == SearchPriorityInteractive {
		return v1.Priority_PRIORITY_INTERACTIVE
	}
	return v1.Priority_PRIORITY_BACKGROUND
}

// StartSearch starts a query over the vault of the client and returns the id of the search at once. An
// equal query of the plugin still running returns its id.
func (c *SearchClient) StartSearch(ctx context.Context, start SearchStart) (string, error) {
	res, err := c.rpc.StartSearch(ctx, newSearchRequest(c, &v1.StartSearchRequest{
		Query: start.Query, Priority: writeSearchPriority(start.Priority), IsBackground: start.IsBackground,
	}))
	if err != nil {
		return "", readSearchCallError(err)
	}
	if res.Msg.GetError() == v1.ErrorCode_ERROR_CODE_LIMIT_EXCEEDED {
		return "", ErrTooManySearches
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return "", err
	}
	return res.Msg.GetId(), nil
}

// GetSearch returns the state of a search the plugin started and one page of its results, waiting up to
// read.Wait, at most 30 seconds, for it to stop.
func (c *SearchClient) GetSearch(ctx context.Context, id string, read SearchRead) (SearchState, error) {
	res, err := c.rpc.GetSearch(ctx, newSearchRequest(c, &v1.GetSearchRequest{
		Id: id, WaitMs: read.Wait.Milliseconds(), PageSize: clampInt32(read.PageSize), PageToken: read.PageToken,
	}))
	if err != nil {
		return SearchState{}, readSearchCallError(err)
	}
	if res.Msg.GetError() == v1.ErrorCode_ERROR_CODE_INVALID {
		return SearchState{}, ErrInvalidPageToken
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return SearchState{}, err
	}
	return readSearchState(res.Msg.GetSearch()), nil
}

func readSearchState(s *v1.SearchState) SearchState {
	problems := make([]SearchProblem, len(s.GetErrors()))
	for i, p := range s.GetErrors() {
		problems[i] = SearchProblem{
			Line: int(p.GetLine()), Column: int(p.GetColumn()), EndLine: int(p.GetEndLine()), EndColumn: int(p.GetEndColumn()),
			Message: p.GetMessage(), Suggestion: p.GetSuggestion(),
		}
	}
	reranks := make([]SearchRerank, len(s.GetReranks()))
	for i, r := range s.GetReranks() {
		reranks[i] = SearchRerank{ConnectionName: r.GetConnectionName(), SentCount: int(r.GetSentCount()), ReusedCount: int(r.GetReusedCount())}
	}
	return SearchState{
		ID: s.GetId(), IsRunning: s.GetIsRunning(), IsPartial: s.GetIsPartial(), IsStale: s.GetIsStale(),
		Step: s.GetProgress().GetStep(), Results: readSearchResults(s.GetResults()), Errors: problems,
		Warnings: s.GetWarnings(), NextPageToken: s.GetNextPageToken(), TotalResults: int(s.GetTotalResults()),
		Reranks: reranks, ScoreKind: s.GetScoreKind(), Floor: s.GetFloor(), IndexGeneration: s.GetIndexGeneration(),
	}
}

// DeleteSearch cancels a running search the plugin started and forgets it.
func (c *SearchClient) DeleteSearch(ctx context.Context, id string) error {
	res, err := c.rpc.DeleteSearch(ctx, newSearchRequest(c, &v1.DeleteSearchRequest{Id: id}))
	if err != nil {
		return readSearchCallError(err)
	}
	return readSearchError(res.Msg.GetError())
}

// ListSearchFunctions returns the functions, variables and stages of the search language.
func (c *SearchClient) ListSearchFunctions(ctx context.Context) (SearchFunctions, error) {
	res, err := c.rpc.ListSearchFunctions(ctx, newSearchRequest(c, &v1.ListSearchFunctionsRequest{}))
	if err != nil {
		return SearchFunctions{}, readSearchCallError(err)
	}
	if err := readSearchError(res.Msg.GetError()); err != nil {
		return SearchFunctions{}, err
	}
	functions := make([]SearchFunction, len(res.Msg.GetFunctions()))
	for i, f := range res.Msg.GetFunctions() {
		functions[i] = SearchFunction{
			Name: f.GetName(), Kind: f.GetKind(), Level: f.GetLevel(), Signatures: f.GetSignatures(),
			Summary: f.GetSummary(), Description: f.GetDescription(), Example: f.GetExample(),
		}
	}
	return SearchFunctions{Functions: functions, Revision: res.Msg.GetRevision()}, nil
}
