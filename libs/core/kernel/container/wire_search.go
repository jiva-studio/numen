package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostsearch"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/embedqueries"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

type textSearch struct {
	VaultReaders vaultport.VaultReaders
	TextReaders  textport.Readers
	Workers      int
}

func newSearch(c textSearch, db *vaultIndex, asking searchsourceport.Source, errorHandler func(error), reportAnswered func()) searchusecase.Search {
	return newSearchOver(c, db.ChunkSearch(), getParticipants(db, asking), errorHandler, reportAnswered)
}

func getParticipants(db *vaultIndex, source searchsourceport.Source) []searchusecase.Participant {
	if source == nil {
		return nil
	}
	return []searchusecase.Participant{searchusecase.NewSource(source, db.ChunkSearch())}
}

func newSearchOver(c textSearch, queries searchusecase.LexicalQueries, sources []searchusecase.Participant, errorHandler func(error), reportAnswered func()) searchusecase.Search {
	searching := searchusecase.New(queries, c.VaultReaders, c.TextReaders, sources, errorHandler, reportAnswered)
	if workers := c.Workers; workers > 0 {
		searching.Extractors = workers
	}
	return searching
}

func newFloorPolicy(db *vaultIndex, hint searchusecase.FloorHint, errorHandler func(error)) searchusecase.FloorPolicy {
	return searchusecase.NewFloorPolicy(db.SearchFloors(), hint, errorHandler)
}

func buildPluginSearch(
	c textSearch, db *vaultIndex, source sourceAccess, errorHandler func(error), clock machineport.Clock,
) hostsearch.Search {
	policy := newFloorPolicy(db, source.floorHint, errorHandler)
	return hostsearch.Search{
		Samples:    db.SearchFloors(),
		Nearest:    searchusecase.NewGetBestSimilarity(db.ChunkSearch()),
		Places:     searchusecase.NewFindNearest(db.ChunkSearch()),
		Progress:   db.ChunkVectors(),
		Stats:      searchusecase.NewStats(db.ChunkVectors(), clock),
		Policy:     policy,
		TextSearch: searchusecase.NewSearchText(newSearch(c, db, source.searchSource, errorHandler, nil)),
		Source:     source.searchSource,
		Queries:    newEmbedQueries(source.queries),
	}
}

func newEmbedQueries(queries searchsourceport.QueryEmbedder) *embedqueries.Embed {
	if queries == nil {
		return nil
	}
	return embedqueries.New(queries)
}
