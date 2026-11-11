package shell

import (
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// vaultLister is the list of vaults an installation keeps, read through a function.
type vaultLister func() ([]domain.Vault, error)

func (l vaultLister) List() ([]domain.Vault, error) { return l() }

func listing(list func() ([]domain.Vault, error)) *registry.List {
	vaults := registry.NewList(vaultLister(list))
	return &vaults
}
