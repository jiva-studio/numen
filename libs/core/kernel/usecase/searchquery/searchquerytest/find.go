// Package searchquerytest builds the search query use case for the tests of several packages.
package searchquerytest

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	usecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

// Deps is what New reads. A nil Clock, Readers or Texts of Stages takes the Clock of Deps and the Readers or Texts of Evaluate.
type Deps struct {
	Parser   usecase.QueryParser
	Clock    machineport.Clock
	Limits   searchquery.Limits
	Evaluate usecase.EvaluateQueryDeps
	Stages   usecase.ApplyStagesDeps
}

// New returns FindSearchResults over the evaluation and the stages the deps describe.
func New(deps Deps) *usecase.FindSearchResults {
	if deps.Stages.Clock == nil {
		deps.Stages.Clock = deps.Clock
	}
	if deps.Stages.Readers == nil {
		deps.Stages.Readers = deps.Evaluate.Readers
	}
	if deps.Stages.Texts == nil {
		deps.Stages.Texts = deps.Evaluate.Texts
	}
	evaluate := usecase.NewEvaluateQuery(deps.Evaluate)
	stages := usecase.NewApplyStages(deps.Stages)
	return usecase.NewFindSearchResults(usecase.FindSearchResultsDeps{
		Parser: deps.Parser, Clock: deps.Clock, Limits: deps.Limits,
		Evaluate: evaluate.Execute, FillDocuments: evaluate.FillDocuments, Stages: stages.Execute,
	})
}
