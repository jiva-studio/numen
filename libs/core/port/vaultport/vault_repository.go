package vaultport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// VaultRepository stores the vaults that the index knows.
type VaultRepository interface {
	// Register creates the vault row that other rows reference. A vault that the
	// index already knows keeps its row and nothing else is written.
	Register(ctx context.Context, vaultID domain.VaultID) error
	// Unregister takes everything the index holds for one vault, and the vault's own
	// row with it.
	Unregister(ctx context.Context, vaultID domain.VaultID) error
}
