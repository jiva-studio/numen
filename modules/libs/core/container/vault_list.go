package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func (c Config) KnownVaults(registry port.VaultRegistry) *vault.KnownVaults {
	known := vault.NewKnownVaults(registry, c.VaultReaders())
	return &known
}

func (c Config) GetVault(registry port.VaultRegistry) *vault.GetVault {
	return vault.NewGetVault(vault.NewFind(registry))
}

func (c Config) PickFolder(dialog port.FolderDialog) *vault.PickFolder {
	return vault.NewPickFolder(dialog)
}
