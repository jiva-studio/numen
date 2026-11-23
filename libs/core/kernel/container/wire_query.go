package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/searchfunctionbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"

	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

// operationResultRows is the most stored results of connection operations the index keeps.
const operationResultRows = 200_000

func newFindSearchResults(
	ports queryPorts, settingsDeps settingsWiring, searchDeps textSearch, db *vaultIndex, source sourceAccess, errorHandler func(error),
) *searchqueryusecase.FindSearchResults {
	limits, stageLimits := readSearchLimits(settingsDeps, errorHandler)
	search := db.ChunkSearch()
	evaluate := searchqueryusecase.NewEvaluateQuery(searchqueryusecase.EvaluateQueryDeps{
		Documents:    search,
		Vectors:      search,
		Places:       search,
		Contents:     db.NoteQueryReads(),
		Links:        db.NoteQueryReads(),
		Functions:    ports.functions,
		Matcher:      wordmatch.NewMatcher(),
		Readers:      searchDeps.VaultReaders,
		Texts:        searchDeps.TextReaders,
		Source:       source.queries,
		Floors:       newFloorPolicy(db, source.floorHint, errorHandler),
		ContentTypes: ports.contentTypes,
	})
	stages := searchqueryusecase.NewApplyStages(searchqueryusecase.ApplyStagesDeps{
		Calls:   ports.calls,
		Rows:    ports.rows,
		Resolve: ports.resolve,
		Results: db.GetOperationResults(operationResultRows),
		Clock:   ports.clock,
		Readers: searchDeps.VaultReaders,
		Texts:   searchDeps.TextReaders,
		Limits:  stageLimits,
	})
	return searchqueryusecase.NewFindSearchResults(searchqueryusecase.FindSearchResultsDeps{
		Parser:        celquery.NewParser(limits),
		Catalog:       ports.getCatalog,
		Clock:         ports.clock,
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

type queryPorts struct {
	functions    searchqueryusecase.SearchFunctions
	contentTypes contenttypeport.ContentTypeTable
	getCatalog      func() searchquerydomain.Catalog
	clock        machineport.Clock
	resolve      connectionport.ResolveConnection
	rows         searchqueryusecase.RerankRows
	calls        connectionport.ModelCalls
}

func openQueryPorts(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) queryPorts {
	ports := queryPorts{functions: searchfunctionbridge.Open(exec.Runtime.Activator), contentTypes: exec.Watch.ContentTypeTable,
		getCatalog: getSearchCatalog(exec.Runtime.Manifests), clock: vaultDeps.Clock, rows: none.RerankRows{}}
	connections, ok := openConnectionDeps(install, exec, settingsDeps, vaultDeps)
	if !ok {
		return ports
	}
	resolve := connection.NewResolve(connections).Execute
	ports.resolve = resolve
	ports.rows = connection.NewRows(resolve, connection.NewListRows(connections, openConnectionRows(install, exec, settingsDeps, vaultDeps)).Execute, nil)
	ports.calls = openModelCalls(install, settingsDeps)
	return ports
}
