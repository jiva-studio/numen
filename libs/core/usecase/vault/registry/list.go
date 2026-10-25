package registry

import (
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// List reports the vaults this installation knows.
type List struct {
	registry vaultport.VaultRegistry
}

// NewList is what the vaults are read out of: the list this installation keeps.
func NewList(registry vaultport.VaultRegistry) List {
	return List{registry: registry}
}

func (u List) Execute() ([]domain.Vault, error) {
	return u.registry.List()
}
