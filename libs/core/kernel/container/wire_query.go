package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/rerankproto"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/operationresult"
	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

// operationResultRows is the most stored results of connection operations the index keeps.
const operationResultRows = 200_000

func newFindSearchResults(
	c pluginsWiring, db *Index, source searchsourceport.Source, errorHandler func(error),
) *searchqueryusecase.FindSearchResults {
	limits, stageLimits := readSearchLimits(c, errorHandler)
	rerank := openRerankDeps(c)
	evaluate := searchqueryusecase.NewEvaluateQuery(searchqueryusecase.EvaluateQueryDeps{
		Chunks:       db.ChunkSearch(),
		Contents:     db.NoteQueryReads(),
		Links:        db.NoteQueryReads(),
		Matcher:      wordmatch.NewMatcher(),
		Readers:      c.search.VaultReaders,
		Texts:        c.search.TextReaders,
		Source:       getQuerySource(source),
		Floors:       newFloorPolicy(db, source, errorHandler),
		ContentTypes: c.search.PluginHost.Watch,
	})
	stages := searchqueryusecase.NewApplyStages(searchqueryusecase.ApplyStagesDeps{
		Calls:   rerank.calls,
		Codec:   rerankproto.Codec{},
		Rows:    rerank.rows,
		Resolve: rerank.resolve,
		Results: db.OperationResults(operationResultRows),
		Clock:   c.vault.Clock,
		Readers: c.search.VaultReaders,
		Texts:   c.search.TextReaders,
		Limits:  stageLimits,
	})
	return searchqueryusecase.NewFindSearchResults(searchqueryusecase.FindSearchResultsDeps{
		Parser:        celquery.NewParser(limits),
		Clock:         c.vault.Clock,
		Limits:        limits,
		Evaluate:      evaluate.Execute,
		FillDocuments: evaluate.FillDocuments,
		Stages:        stages.Execute,
	})
}

func readSearchLimits(c pluginsWiring, errorHandler func(error)) (searchquerydomain.Limits, searchquerydomain.StageLimits) {
	file, err := readCheckedSettings(c.settings)
	if err != nil {
		reportSearchFallback(errorHandler, err)
		return searchquerydomain.DefaultLimits(), searchquerydomain.DefaultStageLimits()
	}
	return file.Search.GetLimits()
}

func newClearOperationResults(db *Index) *operationresult.ClearOperationResults {
	return operationresult.NewClearOperationResults(db.OperationResults(operationResultRows))
}

type rerankDeps struct {
	resolve connectionport.ResolveConnection
	rows    searchqueryusecase.RerankRows
	calls   connectionport.ModelCalls
}

func openRerankDeps(c pluginsWiring) rerankDeps {
	connections, ok := openConnectionDeps(c)
	if !ok {
		return rerankDeps{rows: none.RerankRows{}}
	}
	resolve := connection.NewResolve(connections).Execute
	return rerankDeps{
		resolve: resolve,
		rows:    modelcall.NewRows(resolve, connection.NewListRows(connections, openConnectionRows(c)).Execute, nil),
		calls:   openModelCalls(c),
	}
}

func getQuerySource(source searchsourceport.Source) searchsourceport.QuerySource {
	if query, ok := source.(searchsourceport.QuerySource); ok {
		return query
	}
	return nil
}
