package index

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
)

// chunkQueries is the groups of chunk queries of an index, asked through one value.
type chunkQueries struct {
	*chunk.Files
	*chunk.Search
	*chunk.Vectors
}

func queriesOf(db *DB) chunkQueries {
	return chunkQueries{db.ChunkFiles(), db.ChunkSearch(), db.ChunkVectors()}
}
