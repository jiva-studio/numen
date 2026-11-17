package container

import (
	"context"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	searchrundomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexgeneration"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginmanifest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	searchrunusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchslot"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchstore"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func newIndexGenerations(clock machineport.Clock) *indexgeneration.Generations {
	return indexgeneration.New(strconv.FormatInt(clock.Now().UnixNano(), 36))
}

func emitIndexChanged(generations *indexgeneration.Generations, events *editor.Events) func(context.Context, domain.Vault) {
	return func(ctx context.Context, v domain.Vault) {
		generations.IncreaseGeneration(v.ID)
		events.EmitIndexChanged(ctx, v)
	}
}

func newSearchRuns(w *windowWiring) *searchrunusecase.SearchRuns {
	find := newFindSearchResults(w.a, w.db, w.asking, w.state.ReportFloorError)
	runs := searchrunusecase.NewSearchRuns(searchrunusecase.Deps{
		Store:       searchstore.New(),
		Cancels:     searchcancel.NewSearches(w.cfg.Clock()),
		Tasks:       w.tasks,
		Timer:       w.cfg.Clock(),
		Policy:      searchrundomain.DefaultPolicy(),
		Slots:       searchslot.New(searchrundomain.DefaultPolicy().MaxRunning),
		Generations: w.generations,
		Run:         find.Execute,
		Names:       func(pluginID string) string { return pluginsGetPluginName(w.a, pluginID) },
		Catalog:     getSearchCatalog(getRuntimeManifests(w.a.search.PluginHost.Runtime)),
		MaxResults:  find.Limits.MaxLimit,
	})
	return &runs
}

func getSearchCatalog(manifests pluginmanifest.Manifests) func() searchquerydomain.Catalog {
	return func() searchquerydomain.Catalog {
		return searchquerydomain.MergeCatalog(searchquerydomain.GetBuiltinCatalog(), manifests.ListSearchFunctions())
	}
}

func getListingRevision(catalog func() searchquerydomain.Catalog) func() string {
	return func() string { return searchquerydomain.GetCatalogRevision(catalog()) }
}

func getCoreClaims() manifest.CoreClaims {
	return manifest.CoreClaims{SearchFunctions: searchquerydomain.ListFunctionNames(searchquerydomain.GetBuiltinCatalog())}
}
