package vault

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetVaultToShow returns the vault named, which must be on the list; with none named, the last shown, else the first on the list, else none.
type GetVaultToShow struct {
	registry port.VaultRegistry
	find     Find
	list     List
}

func NewGetVaultToShow(registry port.VaultRegistry) GetVaultToShow {
	return GetVaultToShow{registry: registry, find: NewFind(registry), list: NewList(registry)}
}

func (u GetVaultToShow) Execute(name string) (domain.Vault, error) {
	if name != "" {
		return u.find.Execute(name)
	}
	last, found, err := u.registry.GetLastVault()
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
