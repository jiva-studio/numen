package filesystem

import (
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Readers opens vaults from the filesystem. It is what a use case is handed when
// the vault it works on is chosen while it runs.
type Readers struct{ ServiceDir string }

func (r Readers) Open(v domain.Vault) (port.VaultReader, error) {
	return Open(v.Path, r.ServiceDir)
}
