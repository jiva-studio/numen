package assembly

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"

	vaultport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/vaultcontract"
	windowport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/windowcontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors/embeddingcontract"
)

type vaultSessionAssembly interface {
	GetOpenTabsStore() windowport.OpenTabsStore
	ReadWholeVault(
		ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
	) (vaultreading.ReadWholeVault, error)
	StartVault(
		ctx context.Context, v domain.Vault, rebuild bool,
		onSync func(paths, assets []string, reload bool),
		handleError window.ErrorHandler,
	) (
		read func(ctx context.Context, during func()) (notes int, err error),
		run func(ctx context.Context),
		refresh vaults.Refresh,
		unwatched error,
	)
}

// VaultAssembly is what this window requires to locate, read, change, and watch vaults.
type VaultAssembly interface {
	GetRegistry() vaultport.VaultRegistry
	GetVaultReaders() vaultport.VaultReaders
	GetVaultWriters() vaultport.VaultWriters
	GetVaultIdentity() vaultport.VaultIdentity
	vaultSessionAssembly
}
