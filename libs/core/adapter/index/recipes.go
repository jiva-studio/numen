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

// Compact returns pages that are no longer used to the filesystem.
//
// SQLite reuses a freed page for later rows, so a file that held twice the
// vectors keeps its size until it is rewritten. VACUUM rewrites the whole file,
// so it runs only on an explicit full rebuild of the index.
func (d *DB) Compact(ctx context.Context) error {
	if _, err := sqlwrite.Exec(ctx, d.write, `VACUUM`); err != nil {
		return fmt.Errorf("give back the space the index no longer holds: %w", err)
	}
	return nil
}
