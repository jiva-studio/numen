package chunk

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
)

// Vector is one chunk's embedding in both stored representations.
// `Value` is one byte a dimension, for the rerank. `Coarse` is one bit a
// dimension, read out of `Value`, for the first pass.
// `Hash` addresses the text the vector was computed for, `Recipe` names the model and shape.
// and the shape it was bought under.
type Vector struct {
	Chunk  int64
	Hash   []byte
	Recipe string
	Value  []byte
	Coarse []byte
}

// vectorBatchSize is the number of vectors written in a single transaction before
// committing and checkpointing the WAL, to keep its footprint bounded.
const vectorBatchSize = 1000

// A vector for a chunk that is not in the index is not written, and the
// rest of the group is written.
func (r *Repository) SaveVectors(ctx context.Context, vectors []Vector) error {
	return r.SaveVectorsInBatches(ctx, vectors, vectorBatchSize)
}

// SaveVectorsInBatches is SaveVectors with size vectors written in each transaction.
func (r *Repository) SaveVectorsInBatches(ctx context.Context, vectors []Vector, size int) error {
	if len(vectors) == 0 {
		return nil
	}

	for i := 0; i < len(vectors); i += size {
		end := min(i+size, len(vectors))

		if err := r.saveBatch(ctx, vectors[i:end]); err != nil {
			return err
		}

		// Checkpoint the WAL passively after every batch to keep the file size bounded.
		// A passive checkpoint does not wait for readers and leaves the database unlocked.
		if _, err := sqlwrite.Exec(ctx, r.db, "PRAGMA wal_checkpoint(PASSIVE)"); err != nil {
			return fmt.Errorf("checkpoint wal: %w", err)
		}
	}
	return nil
}

func (r *Repository) saveBatch(ctx context.Context, vectors []Vector) error {
	return sqlwrite.InTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
		for _, v := range vectors {
			// The coarse index takes no conflict clause, so a row that is being
			// replaced is removed first.
			if err := exec(ctx, tx, "delete_vec", v.Chunk); err != nil {
				return err
			}
			if err := exec(ctx, tx, "insert_vec", v.Coarse, v.Chunk); err != nil {
				return err
			}
			if err := exec(ctx, tx, "keep_vector", v.Hash, v.Recipe, v.Value); err != nil {
				return err
			}
		}
		return nil
	})
}

func deleteOrphanVectors(ctx context.Context, tx *sqlwrite.Transaction, texts []string) error {
	for _, hash := range texts {
		if err := exec(ctx, tx, "forget_vector", hash, hash); err != nil {
			return err
		}
	}
	return nil
}
