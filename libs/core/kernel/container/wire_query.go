package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/rerankproto"
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
	c pluginsWiring, db *Index, source searchsourceport.Source, errorHandler func(error),
) *searchqueryusecase.FindSearchResults {
	search := db.ChunkSearch()
	limits, stageLimits := readSearchLimits(c, errorHandler)
	rerank := openRerankDeps(c)
	querySource, _ := source.(searchsourceport.QuerySource)
	deps := searchqueryusecase.FindSearchResultsDeps{
		Parser:       celquery.NewParser(limits),
		Catalog:      getSearchCatalog(getRuntimeManifests(c.search.PluginHost.Runtime)),
		Functions:    searchfunctionbridge.Open(c.search.PluginHost.Runtime.Activator),
		Documents:    search,
		Contents:     db.NoteQueries(),
		Vectors:      search,
		Places:       search,
		Matcher:      wordmatch.NewMatcher(),
		Readers:      c.search.VaultReaders,
		Texts:        c.search.TextReaders,
		Floors:       newFloorPolicy(db, source, errorHandler),
		ContentTypes: c.search.PluginHost.Watch,
		Codec:        rerankproto.Codec{},
		Resolve:      rerank.resolve,
		Calls:        rerank.calls,
		Rows:         rerank.rows,
		Results:      db.OperationResults(operationResultRows),
		Clock:        c.vault.Clock,
		Links:        db.NoteQueries(),
		Limits:       limits,
		StageLimits:  stageLimits,
		Source:       querySource,
	}
	return searchqueryusecase.NewFindSearchResults(deps)
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
