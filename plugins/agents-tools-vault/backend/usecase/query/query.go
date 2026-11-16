// Package query runs queries of the search language through the host and waits a while for their results.
package query

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
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
func (u *UseCase) Start(ctx context.Context, in api.QueryInput) (api.QueryOutput, error) {
	if u.runner == nil {
		return api.QueryOutput{}, errNoQueries
	}
	id, err := u.runner.StartSearch(ctx, host.SearchStart{Query: in.Query, Priority: host.SearchPriorityInteractive})
	if err != nil {
		return api.QueryOutput{}, fmt.Errorf("start the query: %w", err)
	}
	state, err := u.runner.GetSearch(ctx, id, host.SearchRead{Wait: getWait(in.Wait), PageSize: pageSize})
	if err != nil && ctx.Err() != nil {
		return api.QueryOutput{}, errors.Join(ctx.Err(), u.remove(ctx, id))
	}
	if err != nil {
		return api.QueryOutput{}, readError(id, err)
	}
	return writeOutput(state), nil
}

// Results returns the state of a query and one page of its results, waiting for a running query.
func (u *UseCase) Results(ctx context.Context, in api.QueryResultsInput) (api.QueryOutput, error) {
	if u.runner == nil {
		return api.QueryOutput{}, errNoQueries
	}
	state, err := u.runner.GetSearch(ctx, in.ID, host.SearchRead{Wait: getWait(in.Wait), PageSize: pageSize, PageToken: in.PageToken})
	if err != nil {
		return api.QueryOutput{}, readError(in.ID, err)
	}
	return writeOutput(state), nil
}

func (u *UseCase) remove(ctx context.Context, id string) error {
	detached, cancel := context.WithTimeout(context.WithoutCancel(ctx), removeTime)
	defer cancel()
	return u.runner.RemoveSearch(detached, id)
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

func writeOutput(state host.SearchState) api.QueryOutput {
	out := api.QueryOutput{
		ID: state.ID, IsRunning: state.IsRunning, IsPartial: state.IsPartial, IsStale: state.IsStale,
		Progress: api.QueryProgress{Step: state.Step}, Results: make([]api.QueryResult, len(state.Results)),
		Errors: make([]api.QueryProblem, len(state.Errors)), Warnings: append([]string{}, state.Warnings...),
		NextPageToken: state.NextPageToken, TotalResults: state.TotalResults, Message: "",
	}
	for i, r := range state.Results {
		out.Results[i] = writeResult(r)
	}
	for i, p := range state.Errors {
		out.Errors[i] = api.QueryProblem{Line: p.Line, Column: p.Column, Message: p.Message, Suggestion: p.Suggestion}
	}
	if state.IsRunning {
		out.Message = runningMsg
	}
	return out
}

func writeResult(r host.SearchResult) api.QueryResult {
	uri := textplace.FormatFileAddress(r.Path)
	if !r.IsWholeFile {
		uri = textplace.FormatAddress(r.Path, r.Start, r.Length)
	}
	return api.QueryResult{
		Path: r.Path, Kind: r.Kind, Title: r.Title, Place: r.PlaceLabel, Text: r.Text, URI: uri,
		Score: r.Score, IsWholeFile: r.IsWholeFile,
	}
}
