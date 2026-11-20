package container

import (
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostsearch"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
)

type textSearch struct {
	VaultReaders vaultport.VaultReaders
	TextReaders  textdomain.Readers
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

func newFloorPolicy(db *vaultIndex, source searchsourceport.Source, errorHandler func(error)) searchusecase.FloorPolicy {
	return searchusecase.NewFloorPolicy(db.SearchFloors(), getFloorHint(source), errorHandler)
}

func buildPluginSearch(
	c textSearch, db *vaultIndex, source searchsourceport.Source, errorHandler func(error), clock machineport.Clock,
) hostsearch.Search {
	policy := newFloorPolicy(db, source, errorHandler)
	return hostsearch.Search{
		Samples:    db.SearchFloors(),
		Nearest:    searchusecase.NewGetBestSimilarity(db.ChunkSearch()),
		Places:     searchusecase.NewFindNearest(db.ChunkSearch()),
		Progress:   db.ChunkVectors(),
		Stats:      searchusecase.NewStats(db.ChunkVectors(), clock),
		Policy:     policy,
		TextSearch: searchusecase.NewSearchText(newSearch(c, db, source, errorHandler, nil)),
		Source:     source,
		Queries:    newEmbedQueries(source),
	}
}

func newEmbedQueries(source searchsourceport.Source) *vectorscase.EmbedQueries {
	queries, ok := source.(vectorscase.QueryEmbedder)
	if !ok {
		return nil
	}
	return vectorscase.NewEmbedQueries(queries)
}

func getFloorHint(source searchsourceport.Source) searchusecase.FloorHint {
	if advisor, ok := source.(floorAdvisor); ok {
		return advisor.GetFloorHint
	}
	return nil
}
