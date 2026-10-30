package chunk

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
)

// Vector is one chunk's embedding in both representations that are stored.
//
// `Value` is one byte a dimension, for the rerank. `Coarse` is one bit a
// dimension, read out of `Value`, and is what the first pass compares. `Hash`
// addresses the text the vector was bought for, and `Recipe` names the model
// and the shape it was bought under.
type Vector struct {
	Chunk  int64
	Hash   []byte
	Recipe string
	Value  []byte
	Coarse []byte
}

// VectorBatchSize is the number of vectors written in a single transaction before
// committing and checkpointing the WAL, to keep its footprint bounded.
var VectorBatchSize = 1000

// A vector for a chunk that is not in the index is not written, and the
// rest of the group is written.
func (r *Repository) SaveVectors(ctx context.Context, vectors []Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	for i := 0; i < len(vectors); i += VectorBatchSize {
		end := i + VectorBatchSize
		if end > len(vectors) {
			end = len(vectors)
		}

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

// deleteOrphanVectors is called when a source was cut again, which means its
// text was replaced. A source absent from the vault's listing deletes nothing: a
// folder that could not be read looks the same from here as one whose files
// were deleted, and a vector is expensive to recompute.
func deleteOrphanVectors(ctx context.Context, tx *sqlwrite.Transaction, texts []string) error {
	for _, hash := range texts {
		if err := exec(ctx, tx, "forget_vector", hash, hash); err != nil {
			return err
		}
	}
	return nil
}
