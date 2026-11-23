package indexport

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// VaultRepository owns the per-vault rows every other index table references through the vault ID.
type VaultRepository interface {
	// Register creates the vault row that other rows reference. A vault already
	// in the index keeps its row and nothing else is written.
	Register(ctx context.Context, vaultID domain.VaultID) error
	// Unregister takes everything the index holds for one vault, and the vault's own
	// row with it.
	Unregister(ctx context.Context, vaultID domain.VaultID) error
}
