package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/searchfunctionbridge"
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
	rerank rerankDeps, settingsDeps settingsWiring, searchDeps searchWiring, db *Index, source searchsourceport.Source, errorHandler func(error),
) *searchqueryusecase.FindSearchResults {
	limits, stageLimits := readSearchLimits(settingsDeps, errorHandler)
	search := db.ChunkSearch()
	evaluate := searchqueryusecase.NewEvaluateQuery(searchqueryusecase.EvaluateQueryDeps{
		Documents:    search,
		Vectors:      search,
		Places:       search,
		Contents:     db.NoteQueryReads(),
		Links:        db.NoteQueryReads(),
		Functions:    searchfunctionbridge.Open(searchDeps.Runtime.Activator),
		Matcher:      wordmatch.NewMatcher(),
		Readers:      searchDeps.VaultReaders,
		Texts:        searchDeps.TextReaders,
		Source:       getQuerySource(source),
		Floors:       newFloorPolicy(db, source, errorHandler),
		ContentTypes: searchDeps.Watch,
	})
	stages := searchqueryusecase.NewApplyStages(searchqueryusecase.ApplyStagesDeps{
		Calls:   rerank.calls,
		Rows:    rerank.rows,
		Resolve: rerank.resolve,
		Results: db.OperationResults(operationResultRows),
		Clock:   searchDeps.Clock,
		Readers: searchDeps.VaultReaders,
		Texts:   searchDeps.TextReaders,
		Limits:  stageLimits,
	})
	return searchqueryusecase.NewFindSearchResults(searchqueryusecase.FindSearchResultsDeps{
		Parser:        celquery.NewParser(limits),
		Catalog:       getSearchCatalog(getRuntimeManifests(searchDeps.Runtime)),
		Clock:         searchDeps.Clock,
		Limits:        limits,
		Evaluate:      evaluate.Execute,
		FillDocuments: evaluate.FillDocuments,
		Stages:        stages.Execute,
	})
}

func readSearchLimits(settingsDeps settingsWiring, errorHandler func(error)) (searchquerydomain.Limits, searchquerydomain.StageLimits) {
	file, err := readCheckedSettings(settingsDeps)
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

func openRerankDeps(install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) rerankDeps {
	connections, ok := openConnectionDeps(install, settingsDeps, vaultDeps)
	if !ok {
		return rerankDeps{rows: none.RerankRows{}}
	}
	resolve := connection.NewResolve(connections).Execute
	return rerankDeps{
		resolve: resolve,
		rows:    modelcall.NewRows(resolve, connection.NewListRows(connections, openConnectionRows(install, settingsDeps, vaultDeps)).Execute, nil),
		calls:   openModelCalls(install, settingsDeps),
	}
}

func getQuerySource(source searchsourceport.Source) searchsourceport.QuerySource {
	if query, ok := source.(searchsourceport.QuerySource); ok {
		return query
	}
	return nil
}
