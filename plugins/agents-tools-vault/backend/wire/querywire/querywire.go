// Package querywire calls the query use case with the messages of the library query tools.
package querywire

import (
	"context"

	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/query"
)

func StartQuery(ctx context.Context, u *query.UseCase, in api.QueryInput) (api.QueryOutput, error) {
	out, err := u.Start(ctx, query.StartRequest{Query: in.Query, Wait: in.Wait})
	if err != nil {
		return api.QueryOutput{}, err
	}
	return newOutput(out), nil
}

func ReadResults(ctx context.Context, u *query.UseCase, in api.QueryResultsInput) (api.QueryOutput, error) {
	out, err := u.GetResults(ctx, query.ResultsRequest{ID: in.ID, Wait: in.Wait, PageToken: in.PageToken})
	if err != nil {
		return api.QueryOutput{}, err
	}
	return newOutput(out), nil
}

func newOutput(out query.Output) api.QueryOutput {
	results := make([]api.QueryResult, len(out.Results))
	for i, r := range out.Results {
		results[i] = api.QueryResult{
			Path: r.Path, Kind: r.Kind, Title: r.Title, Place: r.Place, Text: r.Text, URI: r.URI,
			Score: r.Score, IsWholeFile: r.IsWholeFile,
		}
	}
	problems := make([]api.QueryProblem, len(out.Errors))
	for i, p := range out.Errors {
		problems[i] = api.QueryProblem{Line: p.Line, Column: p.Column, Message: p.Message, Suggestion: p.Suggestion}
	}
	return api.QueryOutput{
		ID: out.ID, IsRunning: out.IsRunning, IsPartial: out.IsPartial, IsStale: out.IsStale,
		Progress: api.QueryProgress{Step: out.Step}, Results: results, Errors: problems, Warnings: out.Warnings,
		NextPageToken: out.NextPageToken, TotalResults: out.TotalResults, Message: out.Message,
	}
}
