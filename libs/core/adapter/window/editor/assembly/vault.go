package assembly

import (
	"context"

	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// VaultAssembly is what this window asks to locate, read, change, and watch vaults.
type VaultAssembly interface {
	// GetRegistry is the list of vaults this installation holds.
	GetRegistry() vaultport.VaultRegistry

	// GetVaultReaders open a vault for reading and GetVaultWriters for
	// changing.
	GetVaultReaders() vaultport.VaultReaders
	GetVaultWriters() vaultport.VaultWriters

	GetVaultIdentity() vaultport.VaultIdentity

	// GetOpenTabsStore keeps the tabs the window has open on a vault across restarts.
	GetOpenTabsStore() windowport.OpenTabsStore

	// ReadWholeVault is what makes a vault answer: the notes read, the books
	// read, and the vectors made.
	ReadWholeVault(
		ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
	) (vaults.ReadWholeVault, error)

	// StartVault opens a vault behind the window: the walk that brings the
	// cache level with it, and the watch that keeps it level while it is shown.
	//
	// onSync is called each time the two are level again, with the notes that are
	// different now, the files that changed and are not notes, and whether the
	// whole vault has to be looked at again. handleError is called with what
	// went wrong along the way, and with nothing when a later attempt succeeds.
	//
	// It answers with the walk, the watch, what brings a note up to date
	// through this opening, and why the vault is not being followed.
	StartVault(
		ctx context.Context, v domain.Vault, rebuild bool,
		onSync func(paths, assets []string, reload bool),
		handleError windowport.ErrorHandler,
	) (
		read func(ctx context.Context, during func()) (notes int, err error),
		run func(ctx context.Context),
		refresh vaults.Refresh,
		unwatched error,
	)
}
