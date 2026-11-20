package registry

import "github.com/jiva-studio/numen/libs/shared/note/domain"

// LastVault is the vault of the list that was opened most recently.
type LastVault interface {
	GetLastVault() (domain.Vault, bool, error)
}
