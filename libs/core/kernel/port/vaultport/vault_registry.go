package vaultport

import "github.com/jiva-studio/numen/libs/note/domain"

// VaultList lists the vaults the user has added. The list is application state:
// not derivable from any vault, and not stored in one.
type VaultList interface {
	List() ([]domain.Vault, error)
}

// VaultLookup finds one vault of the list by what the user typed.
type VaultLookup interface {
	FindVault(nameOrPath string) (domain.Vault, bool, error)
}

// VaultSaver adds a vault to the list, or updates the one of the same identity.
type VaultSaver interface {
	Save(v domain.Vault) error
}

// VaultRemover takes a vault off the list.
type VaultRemover interface {
	Remove(id domain.VaultID) error
}

// VaultRecorder records which vault was opened last.
type VaultRecorder interface {
	RecordOpened(id domain.VaultID) error
}

// LastVault is the vault of the list that was opened most recently.
type LastVault interface {
	GetLastVault() (domain.Vault, bool, error)
}
