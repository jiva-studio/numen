package vaultreader

import (
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Readers opens vaults from the filesystem. It is what a use case is handed when
// the vault it works on is chosen while it runs.
type Readers struct{ Options vaultconfig.Options }

func (r Readers) Open(v domain.Vault) (vaultport.VaultReader, error) {
	return Open(v.Path, r.Options)
}
