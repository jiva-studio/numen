package chunk_test

import (
	"database/sql"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
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
