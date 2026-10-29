package vaultport

import (
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// VaultIdentity is what turns a folder into a vault: a folder can be looked at
// without being one, and becomes one when it is given an identity that stays
// with it.
type VaultIdentity interface {
	// GetName is the one name this folder has on this machine, whichever route
	// reached it. Two paths leading to one folder are one vault, and which
	// paths those are is the machine's to say: a folder addressed by anything
	// but a path is named by the handle it was given.
	GetName(root string) string
	// CheckReadable reports whether the folder can be read as a vault, writing
	// nothing. Looking and adding are different acts.
	CheckReadable(root string) error
	// GetVaultID returns the identity a folder already carries, and whether it
	// carries one. It never creates an identity, so it checks whether a folder
	// is still the same vault without side effects.
	GetVaultID(root string) (domain.VaultID, bool, error)
	// EnsureIdentity returns the identity the folder carries, creating one if it has
	// none. An existing identity is never replaced: it is what every row in the
	// index points at.
	EnsureIdentity(root string, timestamp time.Time) (domain.VaultID, error)
}
