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
	evaluate := searchqueryusecase.NewEvaluateQuery(searchqueryusecase.EvaluateQueryDeps{
		Documents:    search,
		Vectors:      search,
		Words:        search,
		Places:       search,
		Contents:     db.NoteQueryReads(),
		Matcher:      wordmatch.NewMatcher(),
		Readers:      c.VaultReaders,
		Texts:        c.TextReaders,
		Source:       getQuerySource(source),
		Floors:       newFloorPolicy(db, source, errorHandler),
		ContentTypes: c.PluginHost.Watch,
	})
	return searchqueryusecase.NewFindSearchResults(searchqueryusecase.FindSearchResultsDeps{
		Parser:     celquery.NewParser(searchquerydomain.DefaultLimits()),
		Clock:      c.Clock,
		Evaluate:   evaluate.Execute,
		FillTitles: evaluate.FillTitles,
	})
}

func getQuerySource(source searchsourceport.Source) searchsourceport.QuerySource {
	if query, ok := source.(searchsourceport.QuerySource); ok {
		return query
	}
	return nil
}
