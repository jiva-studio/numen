package port

import "github.com/jiva-studio/numen/modules/libs/core/domain"

// VaultRegistry is the list of vaults the user has added. It is application
// state: not derivable from any vault, and not stored in one.
type VaultRegistry interface {
	All() ([]domain.Vault, error)
	Save(v domain.Vault) error
	Find(nameOrPath string) (domain.Vault, bool, error)
	// Remove takes a vault off the list. The folder and the identity inside it
	// stay as they are, so the same folder can be added again.
	Remove(id string) error
	// Opened records the vault a window is showing. An identity that is not on
	// the list is refused.
	Opened(id string) error
	// Last is the vault opened most recently, and nothing until one has been.
	Last() (domain.Vault, bool, error)
}
