package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// VaultRepository holds the vaults the index knows about.
type VaultRepository interface {
	// Save writes what the vault is called and where it is. The list is what
	// those are on, and this is the index's copy of them.
	Save(ctx context.Context, v domain.Vault) error
	// Register gives the vault a row for other rows to point at, and leaves the
	// name and the path of a vault the index already knows.
	Register(ctx context.Context, v domain.Vault) error
	// Forget takes everything the index holds for one vault, and the vault's own
	// row with it.
	Forget(ctx context.Context, vaultID string) error
}
