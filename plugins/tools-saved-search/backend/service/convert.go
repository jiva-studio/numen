package service

import (
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/domain/resultcache"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/savedsearch"
)

func newSavedOutput(saved savedsearch.SavedSearch) api.SavedOutput {
	return api.SavedOutput{
		Path: saved.Path, URI: saved.URI, ResultCount: saved.ResultCount, Order: api.Order(saved.Order),
		ComputedAt: formatTime(saved.ComputedAt), Warnings: saved.Warnings, Errors: convertProblems(saved.Errors),
	}
}

func convertProblems(problems []resultcache.Problem) []api.Problem {
	if problems == nil {
		return nil
	}
	converted := make([]api.Problem, len(problems))
	for i, problem := range problems {
		converted[i] = api.Problem{
			Line: problem.Line, Column: problem.Column, EndLine: problem.EndLine, EndColumn: problem.EndColumn, Message: problem.Message,
		}
	}
	return converted
}

func convertResults(results []resultcache.Result) []api.Result {
	converted := make([]api.Result, len(results))
	for i, one := range results {
		converted[i] = api.Result{
			Path: one.Path, Title: one.Title, Kind: one.Kind, Place: convertPlace(one.Place), Text: one.Text,
			Start: one.Start, Length: one.Length, Line: one.Line, URI: one.URI, Score: one.Score, IsWholeFile: one.IsWholeFile,
		}
	}
	return converted
}

func convertPlace(place *resultcache.Place) *api.Place {
	if place == nil {
		return nil
	}
	return &api.Place{
		Page: place.Page, PositionSeconds: place.PositionSeconds, EndSeconds: place.EndSeconds, CFI: place.CFI,
		Chapter: place.Chapter,
	}
}
