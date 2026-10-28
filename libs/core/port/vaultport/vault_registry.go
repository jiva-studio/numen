package vaultport

import "github.com/jiva-studio/numen/libs/notes/domain"

type vaultHistoryRegistry interface {
	RecordOpened(id domain.VaultID) error
	GetLastVault() (domain.Vault, bool, error)
}

// VaultRegistry is the list of vaults the user has added. It is application
// state: not derivable from any vault, and not stored in one.
type VaultRegistry interface {
	vaultHistoryRegistry
	List() ([]domain.Vault, error)
	Save(v domain.Vault) error
	FindVault(nameOrPath string) (domain.Vault, bool, error)
	Remove(id domain.VaultID) error
}
