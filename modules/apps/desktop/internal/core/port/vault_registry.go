package port

import "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"

// VaultRegistry is the list of vaults the user has added. It is application
// state: not derivable from any vault, and not stored in one.
type VaultRegistry interface {
	All() ([]domain.Vault, error)
	Save(v domain.Vault) error
	Find(nameOrPath string) (domain.Vault, bool, error)
}
