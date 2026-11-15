package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/rerankproto"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
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
	search := db.ChunkSearch()
	limits, stageLimits := readSearchLimits(c, errorHandler)
	deps := searchqueryusecase.FindSearchResultsDeps{
		Parser:       celquery.NewParser(limits),
		Documents:    search,
		Vectors:      search,
		Words:        search,
		Places:       search,
		Matcher:      wordmatch.NewMatcher(),
		Readers:      c.search.VaultReaders,
		Texts:        c.search.TextReaders,
		Floors:       newFloorPolicy(db, source, errorHandler),
		ContentTypes: c.search.PluginHost.Watch,
		Codec:        rerankproto.Codec{},
		Rows:         none.RerankRows{},
		Results:      db.OperationResults(operationResultRows),
		Clock:        c.vault.Clock,
		Limits:       limits,
		StageLimits:  stageLimits,
	}
	if querySource, ok := source.(searchsourceport.QuerySource); ok {
		deps.Source = querySource
	}
	setRerankConnections(c, &deps)
	return searchqueryusecase.NewFindSearchResults(deps)
}

func setRerankConnections(c pluginsWiring, deps *searchqueryusecase.FindSearchResultsDeps) {
	connections, ok := openConnectionDeps(c)
	if !ok {
		return
	}
	deps.Resolve = connection.NewResolve(connections).Execute
	deps.Rows = modelcall.NewRows(deps.Resolve, connection.NewListRows(connections, openConnectionRows(c)).Execute, nil)
	deps.Calls = openModelCalls(c)
}

func readSearchLimits(c pluginsWiring, errorHandler func(error)) (searchquerydomain.Limits, searchquerydomain.StageLimits) {
	file, err := readCheckedSettings(c.settings)
	if err != nil {
		reportSearchFallback(errorHandler, err)
		return searchquerydomain.DefaultLimits(), searchquerydomain.DefaultStageLimits()
	}
	return file.GetSearchLimits()
}

func newClearOperationResults(db *Index) *operationresult.ClearOperationResults {
	return operationresult.NewClearOperationResults(db.OperationResults(operationResultRows))
}
