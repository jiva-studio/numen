package chunk

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/sqlwrite"
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

// SaveVectors writes both representations of each vector.
//
// They go in batched transactions, and both are written by reading the chunk they
// belong to. A chunk with only one of the two is absent from the coarse pass
// and invisible to the question of what has no vector.
//
// A vector naming a chunk the index no longer holds is written nowhere, and
// what is left of the group is written currently.
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
	tx, err := sqlwrite.Begin(ctx, r.db)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

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
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// deleteOrphanVectors takes out the vectors made for text no chunk holds.
//
// It is asked where a source was cut again, so the text is known to have been
// replaced. A source the vault no longer offers takes nothing with it: a folder
// that could not be read looks the same from here as one whose files were
// deleted, and a vector was bought.
func deleteOrphanVectors(ctx context.Context, tx *sqlwrite.Transaction, texts []string) error {
	for _, hash := range texts {
		if err := exec(ctx, tx, "forget_vector", hash, hash); err != nil {
			return err
		}
	}
	return nil
}
