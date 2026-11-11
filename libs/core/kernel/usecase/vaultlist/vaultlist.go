// Package vaultlist answers what a window shows of the vaults the installation holds: the list, a vault
// added or renamed, and the vault the window switches to.
package vaultlist

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Lister reads the vaults of the installation.
type Lister interface {
	Execute() ([]domain.Vault, error)
}

// Finder finds a vault of the installation by its id.
type Finder interface {
	Execute(nameOrPath string) (domain.Vault, error)
}

// FolderChecker reports whether the folder of a vault is gone.
type FolderChecker interface {
	Execute(v domain.Vault) bool
}

// Adder puts a folder on the list as a vault.
type Adder interface {
	Execute(path, name string) (domain.Vault, error)
}

// Renamer sets the display name of a vault.
type Renamer interface {
	Execute(ctx context.Context, v domain.Vault, name string) (domain.Vault, error)
}

// Shower puts a vault in the window.
type Shower interface {
	ShowVault(ctx context.Context, v domain.Vault) error
}

// Listed is a vault and whether its folder is gone.
type Listed struct {
	domain.Vault
	IsMissing bool
}

// ListVaults lists every vault of the installation.
type ListVaults struct {
	list    Lister
	missing FolderChecker
}

// NewListVaults creates the listing over the registry and the check of folders.
func NewListVaults(list Lister, missing FolderChecker) *ListVaults {
	return &ListVaults{list: list, missing: missing}
}

func (u *ListVaults) Execute() ([]Listed, error) {
	vaults, err := u.list.Execute()
	if err != nil {
		return nil, err
	}
	out := make([]Listed, 0, len(vaults))
	for _, v := range vaults {
		out = append(out, Listed{Vault: v, IsMissing: u.missing.Execute(v)})
	}
	return out, nil
}

// AddVault turns a folder into a vault on the list.
type AddVault struct {
	add     Adder
	missing FolderChecker
}

// NewAddVault creates the adding over the registry and the check of folders.
func NewAddVault(add Adder, missing FolderChecker) *AddVault {
	return &AddVault{add: add, missing: missing}
}

// Execute adds the folder at the path. A name another vault has gets a number appended.
func (u *AddVault) Execute(path, name string) (Listed, error) {
	added, err := u.add.Execute(path, name)
	if err != nil {
		return Listed{}, err
	}
	return Listed{Vault: added, IsMissing: u.missing.Execute(added)}, nil
}

// RenameVault sets the display name of the vault with an id.
type RenameVault struct {
	find    Finder
	rename  Renamer
	missing FolderChecker
}

// NewRenameVault creates the renaming over the registry and the check of folders.
func NewRenameVault(find Finder, rename Renamer, missing FolderChecker) *RenameVault {
	return &RenameVault{find: find, rename: rename, missing: missing}
}

// Execute renames the vault. The folder name on the filesystem is unchanged.
func (u *RenameVault) Execute(ctx context.Context, id, name string) (Listed, error) {
	v, err := u.find.Execute(id)
	if err != nil {
		return Listed{}, err
	}
	renamed, err := u.rename.Execute(ctx, v, name)
	if err != nil {
		return Listed{}, err
	}
	return Listed{Vault: renamed, IsMissing: u.missing.Execute(renamed)}, nil
}

// OpenVault shows another vault in the window.
type OpenVault struct {
	find Finder
	show Shower
}

// NewOpenVault creates the opening over the registry and the window.
func NewOpenVault(find Finder, show Shower) *OpenVault {
	return &OpenVault{find: find, show: show}
}

// Execute shows the vault with the id.
func (u *OpenVault) Execute(ctx context.Context, id string) error {
	v, err := u.find.Execute(id)
	if err != nil {
		return err
	}
	return u.show.ShowVault(ctx, v)
}
