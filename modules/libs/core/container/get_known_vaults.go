package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"

	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

func (c Config) GetKnownVaults(registry vaultport.VaultRegistry) *vault.GetKnownVaults {
	recognized := vault.NewGetKnownVaults(registry, c.VaultReaders())
	return &recognized
}
