package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func (c Config) CheckRoot() *vault.CheckRoot {
	return vault.NewCheckRoot(c.VaultIdentity(), filesystem.HomeFolder{})
}
