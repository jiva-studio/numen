package vault

import (
	"context"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Listed is a vault and whether its folder is gone.
type Listed struct {
	domain.Vault
	IsMissing bool
}

func isFolderMissing(readers vaultport.VaultReaders, v domain.Vault) bool {
	_, err := readers.Open(v)
	return err != nil
}

// ListVaults lists every vault of the installation.
type ListVaults struct {
	vaults  vaultport.VaultList
	readers vaultport.VaultReaders
}

// NewListVaults creates the listing over the list this installation keeps and the readers that tell a gone folder.
func NewListVaults(vaults vaultport.VaultList, readers vaultport.VaultReaders) *ListVaults {
	return &ListVaults{vaults: vaults, readers: readers}
}

func (u *ListVaults) Execute() ([]Listed, error) {
	vaults, err := u.vaults.List()
	if err != nil {
		return nil, err
	}
	out := make([]Listed, 0, len(vaults))
	for _, v := range vaults {
		out = append(out, Listed{Vault: v, IsMissing: isFolderMissing(u.readers, v)})
	}
	return out, nil
}

// AddVault turns a folder into a vault on the list.
type AddVault struct {
	add     Add
	readers vaultport.VaultReaders
}

// NewAddVault creates the adding over Add and the readers that tell a gone folder.
func NewAddVault(add Add, readers vaultport.VaultReaders) *AddVault {
	return &AddVault{add: add, readers: readers}
}

// Execute adds the folder at the path. A name another vault has gets a number appended.
func (u *AddVault) Execute(path, name string) (Listed, error) {
	return u.ExecuteManaged(path, name, "")
}

// RenameVault sets the display name of the vault with an id.
type RenameVault struct {
	lookup  vaultport.VaultLookup
	rename  Rename
	readers vaultport.VaultReaders
}

// NewRenameVault creates the renaming over the list this installation keeps, Rename and the readers that tell a gone folder.
func NewRenameVault(lookup vaultport.VaultLookup, rename Rename, readers vaultport.VaultReaders) *RenameVault {
	return &RenameVault{lookup: lookup, rename: rename, readers: readers}
}

// Execute renames the vault. The folder name on the filesystem is unchanged.
func (u *RenameVault) Execute(ctx context.Context, id, name string) (Listed, error) {
	v, err := vaultdomain.FindByName(id, u.lookup.FindVault)
	if err != nil {
		return Listed{}, err
	}
	renamed, err := u.rename.Execute(ctx, v, name)
	if err != nil {
		return Listed{}, err
	}
	return Listed{Vault: renamed, IsMissing: isFolderMissing(u.readers, renamed)}, nil
}

// ExecuteManaged adds the folder as a vault that only the plugin with the id managedBy writes.
func (u *AddVault) ExecuteManaged(path, name, managedBy string) (Listed, error) {
	added, err := u.add.ExecuteManaged(path, name, managedBy)
	if err != nil {
		return Listed{}, err
	}
	return Listed{Vault: added, IsMissing: isFolderMissing(u.readers, added)}, nil
}
