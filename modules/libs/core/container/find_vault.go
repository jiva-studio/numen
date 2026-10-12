package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func (c Config) FindVault(registry port.VaultRegistry) *vault.Find {
	find := vault.NewFind(registry)
	return &find
}
