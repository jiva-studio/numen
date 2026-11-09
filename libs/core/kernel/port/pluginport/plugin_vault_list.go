package pluginport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// PluginVaultList is the installation's list of vaults and the vault the window shows, changed by
// the plugins that have vault:manage. The window provides it; a host with no window has
// none.
type PluginVaultList interface {
	// AddVault adds the folder at the absolute root. An empty name means the folder's name.
	AddVault(ctx context.Context, root, name string) (domain.Vault, error)
	RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error)
	// RemoveVault removes the vault from the list and from the index. The folder stays.
	RemoveVault(ctx context.Context, id domain.VaultID) error
	// OpenVault shows the vault in the window and returns once the window shows it. A rejected open
	// returns plugin.ErrUnsavedChanges, ErrUnavailable, or an unknown-vault error.
	OpenVault(ctx context.Context, id domain.VaultID) error
	// IsMissing reports whether the vault's folder is missing.
	IsMissing(v domain.Vault) bool
}

// PluginVaultListHost receives the list of vaults exposed to plugins.
type PluginVaultListHost interface {
	// SetPluginVaultList sets the list of vaults exposed to plugins. Nil means none.
	SetPluginVaultList(list PluginVaultList)
}
