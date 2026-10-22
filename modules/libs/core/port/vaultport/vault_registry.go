package vaultport

import "github.com/jiva-studio/numen/modules/libs/notes/domain"

// VaultRegistry is the list of vaults the user has added. It is application
// state: not derivable from any vault, and not stored in one.
type VaultRegistry interface {
	List() ([]domain.Vault, error)
	Save(v domain.Vault) error
	FindVault(nameOrPath string) (domain.Vault, bool, error)
	// Remove takes a vault off the list. The folder and the identity inside it
	// stay as they are, so the same folder can be added again.
	Remove(id domain.VaultID) error
	RecordOpened(id domain.VaultID) error
	// GetLastVault is the vault opened most recently, and nothing until one has been.
	GetLastVault() (domain.Vault, bool, error)
}
