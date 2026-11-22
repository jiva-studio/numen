package searchquery

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// ApplyStagesDeps is what ApplyStages reads. Calls, Rows, Resolve and Results serve the rerank stage,
// Clock times its calls, and a nil Texts reads the text of notes only. A zero Limits takes the defaults.
type ApplyStagesDeps struct {
	Calls   connectionport.ModelCalls
	Rows    RerankRows
	Resolve connectionport.ResolveConnection
	Results OperationResults
	Clock   machineport.Clock
	Readers vaultport.VaultReaders
	Texts   textport.Readers
	Limits  searchquery.StageLimits
}

// ApplyStages runs the stages of a query over its results: rerank, top and sort.
type ApplyStages struct {
	reranker reranker
	readers  vaultport.VaultReaders
	texts    textport.Readers
}

// NewApplyStages creates the use case.
func NewApplyStages(deps ApplyStagesDeps) *ApplyStages {
	limits := deps.Limits
	if limits == (searchquery.StageLimits{}) {
		limits = searchquery.DefaultStageLimits()
	}
	return &ApplyStages{
		reranker: reranker{
			calls: deps.Calls, rows: deps.Rows, resolve: deps.Resolve, results: deps.Results, clock: deps.Clock, limits: limits,
		},
		readers: deps.Readers, texts: deps.Texts,
	}
}
