// Package query runs queries of the search language through the host and waits a while for their results.
package query

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/textplace"
)

const (
	maxWait    = 20 * time.Second
	pageSize   = 20
	removeTime = 5 * time.Second
	runningMsg = "the query is still running: call library_query_results with the id"
)

var errNoQueries = errors.New("the plugin was given no search index")

// StartRequest is a query in the search language and how many seconds to wait for its results; a nil wait is the longest.
type StartRequest struct {
	Query string
	Wait  *float64
}

// ResultsRequest names a query started before, how many seconds to wait for it, and the page to read.
type ResultsRequest struct {
	ID        string
	Wait      *float64
	PageToken string
}

// Result is a passage or a whole file a query found.
type Result struct {
	Path        string
	Kind        string
	Title       string
	Place       string
	Text        string
	URI         string
	Score       *float64
	IsWholeFile bool
}

// Problem is an error of a query.
type Problem struct {
	Line       int
	Column     int
	Message    string
	Suggestion string
}

// Output is the state of a query and one page of its results.
type Output struct {
	ID            string
	IsRunning     bool
	IsPartial     bool
	IsStale       bool
	Step          string
	Results       []Result
	Errors        []Problem
	Warnings      []string
	NextPageToken string
	TotalResults  int
	Message       string
}

// UseCase runs the queries of one vault. The runner is nil when the plugin was given none.
type UseCase struct {
	runner ports.QueryRunner
}

// New creates a UseCase; with a nil runner every call fails.
func New(runner ports.QueryRunner) *UseCase {
	return &UseCase{runner: runner}
}

// Start starts a query and returns its first page, or its state when it runs longer than the wait. A
// call canceled during the wait removes the query.
func (u *UseCase) Start(ctx context.Context, in StartRequest) (Output, error) {
	if u.runner == nil {
		return Output{}, errNoQueries
	}
	id, err := u.runner.StartSearch(ctx, host.SearchStart{Query: in.Query, Priority: host.SearchPriorityInteractive})
	if err != nil {
		return Output{}, fmt.Errorf("start the query: %w", err)
	}
	state, err := u.runner.GetSearch(ctx, id, host.SearchRead{Wait: getWait(in.Wait), PageSize: pageSize})
	if err != nil && ctx.Err() != nil {
		return Output{}, errors.Join(ctx.Err(), u.remove(ctx, id))
	}
	if err != nil {
		return Output{}, readError(id, err)
	}
	return writeOutput(state), nil
}

// GetResults returns the state of a query and one page of its results, waiting for a running query.
func (u *UseCase) GetResults(ctx context.Context, in ResultsRequest) (Output, error) {
	if u.runner == nil {
		return Output{}, errNoQueries
	}
	state, err := u.runner.GetSearch(ctx, in.ID, host.SearchRead{Wait: getWait(in.Wait), PageSize: pageSize, PageToken: in.PageToken})
	if err != nil {
		return Output{}, readError(in.ID, err)
	}
	return writeOutput(state), nil
}

func (u *UseCase) remove(ctx context.Context, id string) error {
	detached, cancel := context.WithTimeout(context.WithoutCancel(ctx), removeTime)
	defer cancel()
	return u.runner.DeleteSearch(detached, id)
}

func getWait(seconds *float64) time.Duration {
	if seconds == nil {
		return maxWait
	}
	wait := time.Duration(*seconds * float64(time.Second))
	return min(max(wait, 0), maxWait)
}

func readError(id string, err error) error {
	if errors.Is(err, host.ErrSearchNotFound) {
		return fmt.Errorf("no query %s: it was removed, or it was not read for 10 minutes", id)
	}
	return fmt.Errorf("read the query %s: %w", id, err)
}

func writeOutput(state host.SearchState) Output {
	out := Output{
		ID: state.ID, IsRunning: state.IsRunning, IsPartial: state.IsPartial, IsStale: state.IsStale,
		Step: state.Step, Results: make([]Result, len(state.Results)),
		Errors: make([]Problem, len(state.Errors)), Warnings: append([]string{}, state.Warnings...),
		NextPageToken: state.NextPageToken, TotalResults: state.TotalResults, Message: "",
	}
	for i, r := range state.Results {
		out.Results[i] = writeResult(r)
	}
	for i, p := range state.Errors {
		out.Errors[i] = Problem{Line: p.Line, Column: p.Column, Message: p.Message, Suggestion: p.Suggestion}
	}
	if state.IsRunning {
		out.Message = runningMsg
	}
	return out
}

func writeResult(r host.SearchResult) Result {
	uri := textplace.FormatFileAddress(r.Path)
	if !r.IsWholeFile {
		uri = textplace.FormatAddress(r.Path, r.Start, r.Length)
	}
	return Result{
		Path: r.Path, Kind: r.Kind, Title: r.Title, Place: r.PlaceLabel, Text: r.Text, URI: uri,
		Score: r.Score, IsWholeFile: r.IsWholeFile,
	}
}
