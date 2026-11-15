package container

import (
	"context"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	searchrundomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexgeneration"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchkept"
	searchrunusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchstore"
	"github.com/jiva-studio/numen/libs/note/domain"
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
		Kept: &searchkept.Searches{
			Store: searchstore.New(), Cancels: searchcancel.NewSearches(w.cfg.Clock()), Tasks: w.tasks,
			Timer: w.cfg.Clock(), Policy: searchrundomain.DefaultPolicy(),
		},
		Generations: w.generations,
		Run:         find.Execute,
		Names:       func(pluginID string) string { return pluginsGetPluginName(w.a, pluginID) },
		Catalog:     searchquerydomain.GetBuiltinCatalog,
		MaxResults:  find.Limits.MaxLimit,
	})
	return &runs
}

func getListingRevision() string {
	return searchquerydomain.GetCatalogRevision(searchquerydomain.GetBuiltinCatalog())
}
