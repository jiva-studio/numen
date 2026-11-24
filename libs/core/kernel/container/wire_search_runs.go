package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	searchrundomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexgeneration"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginmanifest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	queryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchslot"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchstore"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func emitIndexChanged(generations *indexgeneration.Generations, events *editor.Events) func(context.Context, domain.Vault) {
	return func(ctx context.Context, v domain.Vault) {
		generations.BumpGeneration(v.ID)
		events.EmitIndexChanged(ctx, v)
	}
}

type windowSearch struct {
	index  *vaultIndex
	source sourceAccess
	kinds  windowport.TabKinds
}

func newSearchRuns(
	tasks *run.Tasks, generations *indexgeneration.Generations, find *queryusecase.FindSearchResults,
	resolver pluginResolver, manifests pluginmanifest.Manifests, clock installationClock,
) *queryusecase.SearchRuns {
	runs := queryusecase.NewSearchRuns(queryusecase.Deps{
		Store:       searchstore.New(),
		Cancels:     searchcancel.NewSearches(clock),
		Tasks:       tasks,
		Clock:       clock,
		CreateTimer: clock.After,
		Policy:      searchrundomain.DefaultPolicy(),
		Slots:       searchslot.New(searchrundomain.DefaultPolicy().MaxRunning),
		Generations: generations,
		Run:         find.Execute,
		Names:       getPluginNames(resolver),
		GetCatalog:  getSearchCatalog(manifests),
		MaxResults:  find.Limits.MaxLimit,
	})
	return &runs
}

func getSearchCatalog(manifests pluginmanifest.Manifests) func() searchquerydomain.Catalog {
	return func() searchquerydomain.Catalog {
		return searchquerydomain.MergeCatalog(searchquerydomain.GetBuiltinCatalog(), manifests.ListSearchFunctions())
	}
}
