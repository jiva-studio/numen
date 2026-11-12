package windowsession

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// VaultAdd adds the folder at the absolute root to the vaults; an empty name means the folder's name.
type VaultAdd interface {
	Execute(root, name string) (domain.Vault, error)
}

// VaultFind finds the vault the id names.
type VaultFind interface {
	Execute(id string) (domain.Vault, error)
}

type VaultRename interface {
	Execute(ctx context.Context, v domain.Vault, name string) (domain.Vault, error)
}

// VaultRemove removes a vault from the list and from the index, and erases its folder when asked to.
type VaultRemove interface {
	Execute(ctx context.Context, id string, current domain.VaultID, isErasing bool) error
}

// FolderCheck is whether the folder of a vault is missing.
type FolderCheck interface {
	Execute(v domain.Vault) bool
}

// VaultOpening shows a vault in the window on behalf of a plugin.
type VaultOpening interface {
	OpenPluginVault(ctx context.Context, v domain.Vault) error
}

// VaultEdits are the operations that change the list of vaults.
type VaultEdits struct {
	Add    VaultAdd
	Rename VaultRename
	Remove VaultRemove
}

// VaultOps are the operations on the list of vaults that plugins reach.
type VaultOps struct {
	Edit    VaultEdits
	Find    VaultFind
	Missing FolderCheck
	Open    VaultOpening
}

// PluginVaults lets the plugins that have vault:manage change the list of vaults and the vault the
// window shows. It uses the operations of the window, so a plugin and the user see the same errors.
type PluginVaults struct {
	current currentVault
	ops     VaultOps
}

// NewPluginVaults makes the list of vaults plugins change, over the runtime of the window.
func NewPluginVaults(rt *Runtime, ops VaultOps) PluginVaults {
	return PluginVaults{current: rt, ops: ops}
}

// AddVault adds the folder at the absolute root. An empty name means the folder's name.
func (l PluginVaults) AddVault(_ context.Context, root, name string) (domain.Vault, error) {
	return l.ops.Edit.Add.Execute(root, name)
}

func (l PluginVaults) RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error) {
	v, err := l.ops.Find.Execute(string(id))
	if err != nil {
		return domain.Vault{}, err
	}
	return l.ops.Edit.Rename.Execute(ctx, v, name)
}

// RemoveVault removes the vault from the list and from the index, and does not delete its folder.
func (l PluginVaults) RemoveVault(ctx context.Context, id domain.VaultID) error {
	return l.ops.Edit.Remove.Execute(ctx, string(id), l.current.GetCurrentVault().ID, false)
}

// OpenVault shows the vault the id names in the window.
func (l PluginVaults) OpenVault(ctx context.Context, id domain.VaultID) error {
	v, err := l.ops.Find.Execute(string(id))
	if err != nil {
		return err
	}
	return l.ops.Open.OpenPluginVault(ctx, v)
}

// IsMissing reports whether the folder of the vault is missing.
func (l PluginVaults) IsMissing(v domain.Vault) bool { return l.ops.Missing.Execute(v) }
