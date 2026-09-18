package index

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/writing"
)

// ForgetOtherRecipes takes out every vector kept under a recipe other than the
// one in use, and says how many went.
//
// A vector is addressed by the text it was made from and the recipe it was made
// under, so a recipe that moved leaves the vectors of the old one on disk beside
// the new. They stay there for a person who sets the old model back, and this is
// asked only where a person has said to build the index again.
func (d *DB) ForgetOtherRecipes(ctx context.Context, recipe string) (int64, error) {
	if recipe == "" {
		return 0, nil
	}
	res, err := writing.Exec(ctx, d.write, `DELETE FROM vectors WHERE recipe <> ?`, recipe)
	if err != nil {
		return 0, fmt.Errorf("forget the vectors of every other recipe: %w", err)
	}
	return res.RowsAffected()
}

// Compact hands the pages nothing holds any more back to the filesystem.
//
// SQLite keeps a freed page for the next row that wants one, so a file that
// held twice the vectors goes on being that size until it is written out again.
// VACUUM is that writing out, and it is the whole file: it is asked where a
// person has already asked for the index to be built again.
func (d *DB) Compact(ctx context.Context) error {
	if _, err := writing.Exec(ctx, d.write, `VACUUM`); err != nil {
		return fmt.Errorf("give back the space the index no longer holds: %w", err)
	}
	return nil
}
