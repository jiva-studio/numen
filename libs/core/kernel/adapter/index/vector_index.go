package index

import (
	"database/sql"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
)

// VectorIndex is the store of vectors as a whole: its width, the recipes it
// holds vectors of and the shortest chunk that gets a vector. It writes, so it
// takes the write pool.
type VectorIndex struct {
	write   *sql.DB
	minimum *chunk.Minimum
}

// VectorIndex returns the maintenance operations of the vector store of this index.
func (d *DB) VectorIndex() *VectorIndex { return &VectorIndex{write: d.write, minimum: &d.minimum} }
