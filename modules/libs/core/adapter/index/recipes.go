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
// under, so a model, a width, a pooling or a quantisation that changed leaves
// the vectors of the old one on disk beside the new. They stay there on
// purpose: a person who sets the old model back finds every one of them and
// buys nothing a second time. The cost of that is a store that grows by its own
// size each time a recipe moves and never shrinks.
//
// This is how the space comes back, and it is asked only where a person has
// said to build the index again.
func (db *DB) ForgetOtherRecipes(ctx context.Context, recipe string) (int64, error) {
	if recipe == "" {
		return 0, nil
	}
	res, err := writing.Exec(ctx, db.write, `DELETE FROM vectors WHERE recipe <> ?`, recipe)
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
func (db *DB) Compact(ctx context.Context) error {
	if _, err := writing.Exec(ctx, db.write, `VACUUM`); err != nil {
		return fmt.Errorf("give back the space the index no longer holds: %w", err)
	}
	return nil
}
