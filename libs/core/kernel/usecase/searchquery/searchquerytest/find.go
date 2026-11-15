// Package searchquerytest builds the search query use case for the tests of several packages.
package searchquerytest

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

// Deps is what New reads.
type Deps struct {
	Parser   searchqueryusecase.QueryParser
	Clock    machineport.Clock
	Evaluate searchqueryusecase.EvaluateQueryDeps
}

// New returns FindSearchResults over the evaluation the deps describe.
func New(deps Deps) *searchqueryusecase.FindSearchResults {
	evaluate := searchqueryusecase.NewEvaluateQuery(deps.Evaluate)
	return searchqueryusecase.NewFindSearchResults(searchqueryusecase.FindSearchResultsDeps{
		Parser: deps.Parser, Clock: deps.Clock, Evaluate: evaluate.Execute, FillTitles: evaluate.FillTitles,
	})
}
