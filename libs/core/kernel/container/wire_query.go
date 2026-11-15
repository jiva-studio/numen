package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

func newFindSearchResults(
	c searchWiring, db *Index, source searchsourceport.Source, errorHandler func(error),
) *searchqueryusecase.FindSearchResults {
	search := db.ChunkSearch()
	deps := searchqueryusecase.FindSearchResultsDeps{
		Parser:       celquery.NewParser(searchquerydomain.DefaultLimits()),
		Documents:    search,
		Contents:     db.NoteContents(),
		Clock:        c.Clock,
		Vectors:      search,
		Words:        search,
		Places:       search,
		Matcher:      wordmatch.NewMatcher(),
		Readers:      c.VaultReaders,
		Texts:        c.TextReaders,
		Floors:       newFloorPolicy(db, source, errorHandler),
		ContentTypes: c.PluginHost.Watch,
	}
	if querySource, ok := source.(searchsourceport.QuerySource); ok {
		deps.Source = querySource
	}
	return searchqueryusecase.NewFindSearchResults(deps)
}
