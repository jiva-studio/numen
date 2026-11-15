package hostendpoint

import (
	"fmt"
	"math"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	searchrunusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchrun"
)

func writeSearchState(page searchrunusecase.Page) (*v1.SearchState, error) {
	r := page.Record
	results := make([]*v1.SearchResult, len(r.Results.Results))
	for i, one := range r.Results.Results {
		result, err := writeQueryResult(one)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return &v1.SearchState{
		Id: string(r.ID), IsRunning: r.IsRunning, IsPartial: r.IsPartial, IsStale: page.IsStale,
		Progress: &v1.SearchProgress{Step: searchrun.GetStepName(r.Step)},
		Results:  results, Errors: writeSearchProblems(r.Errors), Warnings: r.Results.Warnings,
		NextPageToken: page.NextPageToken, TotalResults: int32(min(page.TotalResults, math.MaxInt32)),
		Reranks: writeSearchReranks(r.Results.Reranks), ScoreKind: getScoreKindName(r.Results.ScoreKind),
		Floor: r.Results.Floor, IndexGeneration: r.IndexGeneration,
	}, nil
}

func writeQueryResult(one searchquery.SearchResult) (*v1.SearchResult, error) {
	end := one.Start + one.Length
	if one.Start < 0 || one.Length < 0 || end > math.MaxInt32 || one.MatchLine < 0 || one.MatchLine > math.MaxInt32 {
		return nil, connect.NewError(connect.CodeOutOfRange, fmt.Errorf("the result of %s at %d+%d, line %d, exceeds the 32-bit range", one.Document.Path, one.Start, one.Length, one.MatchLine))
	}
	result := &v1.SearchResult{
		Path: one.Document.Path, Text: one.Text, PlaceLabel: one.PlaceLabel, MatchLine: int32(one.MatchLine),
		Selector: string(one.Selector), Kind: one.Document.Kind, Title: one.Document.Title, Score: one.Score,
		IsWholeFile: one.IsWholeFile(),
	}
	if !one.IsWholeFile() {
		result.ByteRange = &v1.Range{Start: int32(one.Start), End: int32(end)}
	}
	return result, nil
}

func writeSearchProblems(problems []searchrun.Problem) []*v1.SearchProblem {
	out := make([]*v1.SearchProblem, len(problems))
	for i, p := range problems {
		out[i] = &v1.SearchProblem{
			Line: int32(min(p.Line, math.MaxInt32)), Column: int32(min(p.Column, math.MaxInt32)),
			Message: p.Message, Suggestion: p.Suggestion,
		}
	}
	return out
}

func writeSearchReranks(reports []searchquery.RerankReport) []*v1.SearchRerank {
	out := make([]*v1.SearchRerank, len(reports))
	for i, r := range reports {
		out[i] = &v1.SearchRerank{
			ConnectionName: r.ConnectionName, SentCount: int32(min(r.SentCount, math.MaxInt32)),
			ReusedCount: int32(min(r.ReusedCount, math.MaxInt32)),
		}
	}
	return out
}

func getScoreKindName(kind searchquery.ScoreKind) string {
	switch kind {
	case searchquery.ScoreMeaning:
		return "meaning"
	case searchquery.ScoreWords:
		return "words"
	case searchquery.ScoreFused:
		return "fused"
	case searchquery.ScoreRerank:
		return "rerank"
	case searchquery.ScoreNone:
	}
	return "none"
}

func writeSearchFunctions(catalog searchquery.Catalog) []*v1.SearchFunction {
	out := make([]*v1.SearchFunction, len(catalog.Functions))
	for i, f := range catalog.Functions {
		out[i] = &v1.SearchFunction{
			Name: f.Name, Kind: getFunctionKind(f), Level: getLevelName(f), Signatures: searchquery.FormatSignatures(f),
			Description: f.Description, Example: f.Example,
		}
	}
	return out
}

func getFunctionKind(f searchquery.Function) string {
	switch {
	case f.IsStage:
		return "stage"
	case f.IsVariable:
		return "variable"
	}
	return "function"
}

func getLevelName(f searchquery.Function) string {
	switch {
	case f.IsStage:
		return ""
	case f.Level == searchquery.LevelDocument:
		return "document"
	}
	return "chunk"
}
