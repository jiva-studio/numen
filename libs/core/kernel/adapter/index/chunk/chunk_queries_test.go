package chunk_test

import (
	"context"
	"database/sql"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// chunkQueries is the groups of chunk queries of an index, asked through one value.
type chunkQueries struct {
	*chunk.Files
	*chunk.Search
	*chunk.Vectors
}

func queriesOf(db *index.DB) chunkQueries {
	return chunkQueries{db.ChunkFiles(), db.ChunkSearch(), db.ChunkVectors()}
}

func queriesOfRaw(raw *sql.DB) chunkQueries {
	minimum := &chunk.Minimum{}
	return chunkQueries{chunk.NewFiles(raw), chunk.NewSearch(raw), chunk.NewVectors(raw, minimum)}
}

// FindNearest runs the search by meaning over the index.
func (c chunkQueries) FindNearest(ctx context.Context, vaultID domain.VaultID, q sourcedomain.NearestQuery) ([]domain.Passage, error) {
	return search.NewFindNearest(c.Search).Execute(ctx, vaultID, q)
}

// GetBestSimilarity reads the similarity of the passage nearest the query.
func (c chunkQueries) GetBestSimilarity(ctx context.Context, vaultID domain.VaultID, recipe string, query []float32) (float64, error) {
	return search.NewGetBestSimilarity(c.Search).Execute(ctx, vaultID, recipe, query)
}
