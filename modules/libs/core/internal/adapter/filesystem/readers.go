package filesystem

import (
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// VaultReaders opens vaults from the filesystem. It is what a use case is handed when
// the vault it works on is chosen while it runs.
type VaultReaders struct{ Options Options }

func (r VaultReaders) Open(v domain.Vault) (vaultport.VaultReader, error) {
	return Open(v.Path, r.Options)
}
