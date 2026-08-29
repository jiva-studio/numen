package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// VaultRepository holds the vaults the index knows about.
type VaultRepository interface {
	Save(ctx context.Context, v domain.Vault) error
	// Forget takes everything the index holds for one vault, and the vault's own
	// row with it.
	Forget(ctx context.Context, vaultID string) error
}
