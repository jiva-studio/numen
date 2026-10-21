package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func (c Config) GetKnownVaults(registry port.VaultRegistry) *vault.GetKnownVaults {
	recognized := vault.NewGetKnownVaults(registry, c.VaultReaders())
	return &recognized
}
