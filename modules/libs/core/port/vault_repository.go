package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// VaultRepository holds the vaults the index knows about.
type VaultRepository interface {
	// Register gives the vault a row for other rows to point at. A vault the
	// index already knows keeps the row it has; there is nothing else here for
	// it to write.
	Register(ctx context.Context, vaultID domain.VaultID) error
	// Forget takes everything the index holds for one vault, and the vault's own
	// row with it.
	Forget(ctx context.Context, vaultID string) error
}
