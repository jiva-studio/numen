// Package searchquerytest builds the search query use case for the tests of several packages.
package searchquerytest

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

// Deps is what New reads. A nil Clock, Readers or Texts of Stages takes the Clock of Deps and the Readers or Texts of Evaluate.
type Deps struct {
	Parser     searchqueryusecase.QueryParser
	GetCatalog func() searchquery.Catalog
	Clock      machineport.Clock
	Limits     searchquery.Limits
	Evaluate   searchqueryusecase.EvaluateQueryDeps
	Stages     searchqueryusecase.ApplyStagesDeps
}

// New returns FindSearchResults over the evaluation and the stages the deps describe.
func New(deps Deps) *searchqueryusecase.FindSearchResults {
	if deps.Stages.Clock == nil {
		deps.Stages.Clock = deps.Clock
	}
	if deps.Stages.Readers == nil {
		deps.Stages.Readers = deps.Evaluate.Readers
	}
	if deps.Stages.Texts == nil {
		deps.Stages.Texts = deps.Evaluate.Texts
	}
	evaluate := searchqueryusecase.NewEvaluateQuery(deps.Evaluate)
	stages := searchqueryusecase.NewApplyStages(deps.Stages)
	return searchqueryusecase.NewFindSearchResults(searchqueryusecase.FindSearchResultsDeps{
		Parser: deps.Parser, GetCatalog: deps.GetCatalog, Clock: deps.Clock, Limits: deps.Limits,
		Evaluate: evaluate.Execute, FillDocuments: evaluate.FillDocuments, RunStages: stages.Execute,
	})
}
