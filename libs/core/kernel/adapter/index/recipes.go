package index

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
)

// DeleteVectorsOfOtherRecipes deletes every vector stored under a recipe other than the
// current one and returns the number deleted.
// A vector is addressed by its source text and its recipe, so a changed recipe
// leaves the old vectors on disk; they stay for a user who restores the old
// model and are deleted only on an explicit full rebuild.
func (d *VectorIndex) DeleteVectorsOfOtherRecipes(ctx context.Context, recipe string) (int64, error) {
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
func (d *VectorIndex) Compact(ctx context.Context) error {
	if _, err := sqlwrite.Exec(ctx, d.write, `VACUUM`); err != nil {
		return fmt.Errorf("give back the space the index no longer holds: %w", err)
	}
	return nil
}

// ApplyMinChunkLength sets the shortest small chunk that gets a vector and returns how many coarse rows it removed.
// Shorter chunks lose their coarse row and stored vector under every recipe and
// stay searchable by their words. A large chunk is not touched.
// Asked again with the length in force it removes nothing; a process that starts
// with a length above zero removes on its first call, as it cannot know what an earlier process left.
func (d *VectorIndex) ApplyMinChunkLength(ctx context.Context, length int) (int64, error) {
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

func (d *VectorIndex) deleteShortVectors(ctx context.Context, length int) (removed int64, err error) {
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
