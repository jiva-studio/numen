package vault

import (
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// List reports the vaults this installation knows.
type List struct {
	registry port.VaultRegistry
}

// NewList is what the vaults are read out of: the list this installation keeps.
func NewList(registry port.VaultRegistry) List {
	return List{registry: registry}
}

func (u List) Execute() ([]domain.Vault, error) {
	return u.registry.List()
}
