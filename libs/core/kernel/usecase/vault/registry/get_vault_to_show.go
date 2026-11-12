package registry

import (
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// GetVaultToShow returns the vault named, which must be on the list; with none named, the last shown, else the first on the list, else none.
type GetVaultToShow struct {
	last vaultport.LastVault
	find Find
	list List
}

func NewGetVaultToShow(lookup vaultport.VaultLookup, vaults vaultport.VaultList, last vaultport.LastVault) GetVaultToShow {
	return GetVaultToShow{last: last, find: NewFind(lookup), list: NewList(vaults)}
}

func (u GetVaultToShow) Execute(name string) (domain.Vault, error) {
	if name != "" {
		return u.find.Execute(name)
	}
	last, found, err := u.last.GetLastVault()
	if err != nil {
		return domain.Vault{}, err
	}
	if found {
		return last, nil
	}
	target, err := u.list.Execute()
	if err != nil {
		return domain.Vault{}, err
	}
	if len(target) > 0 {
		return target[0], nil
	}
	return domain.Vault{}, nil
}
