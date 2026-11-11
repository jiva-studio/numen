package registry

import (
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// List returns the vaults registered in this installation.
type List struct {
	vaults vaultport.VaultList
}

// NewList is what the vaults are read out of: the list this installation keeps.
func NewList(vaults vaultport.VaultList) List {
	return List{vaults: vaults}
}

func (u List) Execute() ([]domain.Vault, error) {
	return u.vaults.List()
}

// Listing looks up the vaults on the list.
type Listing struct {
	List        List
	Find        Find
	FolderCheck FolderCheck
}
