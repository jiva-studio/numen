package vault

import (
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// List reports the vaults this installation knows.
type List struct {
	Registry port.VaultRegistry
}

func (u List) Execute() ([]domain.Vault, error) {
	return u.Registry.All()
}
