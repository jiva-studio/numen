package vault

import (
	"context"

	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ListAccess is the List a window gives the plugins that have vault:manage, made of the use cases the window itself
// changes the list with, so a plugin and the user see the same errors.
type ListAccess struct {
	add     *vaults.AddVault
	rename  *vaults.RenameVault
	remove  vaults.Remove
	open    windowsession.OpenPluginVault
	missing registry.FolderCheck
	getCurrentVault func() domain.VaultID
}

// NewListAccess makes the list over the use cases, and current, the vault the window shows.
func NewListAccess(
	add *vaults.AddVault, rename *vaults.RenameVault, remove vaults.Remove, open windowsession.OpenPluginVault,
	missing registry.FolderCheck, getCurrentVault func() domain.VaultID,
) ListAccess {
	return ListAccess{add: add, rename: rename, remove: remove, open: open, missing: missing, getCurrentVault: getCurrentVault}
}

// AddVault adds the folder at the absolute root. An empty name means the folder's name.
func (l ListAccess) AddVault(_ context.Context, root, name string) (domain.Vault, error) {
	added, err := l.add.Execute(root, name)
	return added.Vault, err
}

func (l ListAccess) RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error) {
	renamed, err := l.rename.Execute(ctx, string(id), name)
	return renamed.Vault, err
}

// RemoveVault removes the vault from the list and from the index, and does not delete its folder.
func (l ListAccess) RemoveVault(ctx context.Context, id domain.VaultID) error {
	return l.remove.Execute(ctx, string(id), l.getCurrentVault(), false)
}

// OpenVault shows the vault the id names in the window.
func (l ListAccess) OpenVault(ctx context.Context, id domain.VaultID) error {
	return l.open.Execute(ctx, id)
}

// IsMissing reports whether the folder of the vault is missing.
func (l ListAccess) IsMissing(v domain.Vault) bool { return l.missing.Execute(v) }
