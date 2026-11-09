package indexing

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/indexport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// GetProgress queries the index progress for a vault and recipe.
func GetProgress(ctx context.Context, progress indexport.IndexProgress, vaultID domain.VaultID, recipe string) (total, embedded int64, err error) {
	return progress.GetProgress(ctx, vaultID, recipe)
}
