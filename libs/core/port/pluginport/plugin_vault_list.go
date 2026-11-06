package pluginport

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrUnsavedChanges is what OpenVault returns while the window asks the user about text a page
// cannot write without them. The window keeps its vault.
var ErrUnsavedChanges = errors.New("the user is asked about unsaved changes in the window; ask again after they answer")

// PluginVaultList is the installation's list of vaults and the vault the window shows, as the
// plugins that hold vault:manage change them. The window provides it; a host with no window has
// none.
type PluginVaultList interface {
	// AddVault adds the folder at the absolute root. An empty name takes the folder's name.
	AddVault(ctx context.Context, root, name string) (domain.Vault, error)
	RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error)
	// RemoveVault takes the vault off the list and out of the index. The folder stays.
	RemoveVault(ctx context.Context, id domain.VaultID) error
	// OpenVault shows the vault in the window and returns once the window shows it. A rejected open
	// is the error: ErrUnsavedChanges, ErrUnavailable, or an unknown vault.
	OpenVault(ctx context.Context, id domain.VaultID) error
	// IsMissing reports whether nothing is at the vault's folder.
	IsMissing(v domain.Vault) bool
}

// PluginVaultListHost receives the list of vaults that plugins change.
type PluginVaultListHost interface {
	// SetPluginVaultList is the list of vaults the plugins change. Nil is none.
	SetPluginVaultList(list PluginVaultList)
}
