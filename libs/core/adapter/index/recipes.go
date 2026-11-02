package index

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
)

// DeleteOtherRecipeVectors deletes every vector stored under a recipe other than the
// current one and returns the number deleted.
//
// A vector is addressed by its source text and its recipe, so a changed recipe leaves
// the vectors of the old recipe on disk next to the new ones. They remain for a user
// who restores the old model, and are deleted only on an explicit full rebuild
// of the index.
func (d *DB) DeleteOtherRecipeVectors(ctx context.Context, recipe string) (int64, error) {
	if recipe == "" {
		return 0, nil
	}
	res, err := sqlwrite.Exec(ctx, d.write, `DELETE FROM vectors WHERE recipe <> ?`, recipe)
	if err != nil {
		return 0, fmt.Errorf("forget the vectors of every other recipe: %w", err)
	}
	return res.RowsAffected()
}

// Compact returns unused pages to the filesystem.
//
// SQLite reuses a freed page for later rows, so a file that contained twice the
// vectors retains its size until it is rewritten. VACUUM rewrites the whole file,
// so it runs only on an explicit full rebuild of the index.
func (d *DB) Compact(ctx context.Context) error {
	if _, err := sqlwrite.Exec(ctx, d.write, `VACUUM`); err != nil {
		return fmt.Errorf("give back the space the index no longer holds: %w", err)
	}
	return nil
}

// ApplyMinChunkLength sets the shortest small chunk that gets a vector and removes the vectors of
// the small chunks that are shorter. It returns how many coarse rows it removed.
//
// A chunk below the length is left out of the chunks to embed and of the counts of chunks that can
// carry a vector, and it stays searchable by its words. The coarse row of the chunk and the vector
// kept for its text go, under every recipe: text this short is not worth a vector under any model.
// A large chunk carries no vector and is not touched.
//
// Asked again with the length already in force it removes nothing, because a chunk cut or changed
// after the first call is given no vector. A process that starts with a length above zero removes
// on its first call, as it cannot know what an earlier process left.
func (d *DB) ApplyMinChunkLength(ctx context.Context, length int) (int64, error) {
	length = max(length, 0)
	before := d.minimum.Swap(int64(length))
	if before == int64(length) || length == 0 {
		return 0, nil
	}
	removed, err := d.deleteShortVectors(ctx, length)
	if err != nil {
		d.minimum.Swap(before)
		return 0, err
	}
	return removed, nil
}

func (d *DB) deleteShortVectors(ctx context.Context, length int) (removed int64, err error) {
	err = sqlwrite.InTransaction(ctx, d.write, func(tx *sqlwrite.Transaction) error {
		coarse, err := tx.ExecContext(ctx,
			`DELETE FROM chunks_vec WHERE chunk_id IN (SELECT id FROM chunks WHERE parent_id IS NOT NULL AND length < ?)`, length)
		if err != nil {
			return fmt.Errorf("forget the coarse rows of the chunks shorter than %d: %w", length, err)
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM vectors WHERE hash IN (SELECT unhex(hash) FROM chunks WHERE parent_id IS NOT NULL AND length < ?)`, length); err != nil {
			return fmt.Errorf("forget the vectors of the chunks shorter than %d: %w", length, err)
		}
		removed, err = coarse.RowsAffected()
		return err
	})
	return removed, err
}
