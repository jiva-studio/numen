package pluginport

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrVaultAsking is what OpenVault returns when a tab holds changes the person has to decide on, so
// the window keeps its vault.
var ErrVaultAsking = errors.New("a tab holds changes the person has to decide on")

// PluginVaultList is the installation's list of vaults and the vault the window shows, as the
// plugins that hold vault:manage change them. The window provides it; a host with no window has
// none.
type PluginVaultList interface {
	// AddVault adds the folder at the absolute root. An empty name takes the folder's name.
	AddVault(ctx context.Context, root, name string) (domain.Vault, error)
	RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error)
	// RemoveVault takes the vault off the list and out of the index. The folder stays.
	RemoveVault(ctx context.Context, id domain.VaultID) error
	// OpenVault decides whether the window can show the vault and returns the change of vault, which
	// the caller runs. A refusal is the error: ErrVaultAsking, or an unknown vault.
	OpenVault(ctx context.Context, id domain.VaultID) (swap func(ctx context.Context) error, err error)
	// IsMissing reports whether nothing is at the vault's folder.
	IsMissing(v domain.Vault) bool
}

// PluginVaultListHost receives the list of vaults that plugins change.
type PluginVaultListHost interface {
	// SetPluginVaultList is the list of vaults the plugins change. Nil is none.
	SetPluginVaultList(list PluginVaultList)
}
