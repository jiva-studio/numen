package port

import (
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// VaultIdentity is what turns a folder into a vault: a folder can be looked at
// without being one, and becomes one when it is given an identity that stays
// with it.
type VaultIdentity interface {
	// Readable reports whether the folder can be read as a vault, writing
	// nothing. Looking and adding are different acts.
	Readable(root string) error
	// Of returns the identity a folder already carries, and whether it carries
	// one. It never creates an identity, which is what makes it usable for
	// asking whether a folder is still the vault it used to be.
	Of(root string) (domain.VaultID, bool, error)
	// Ensure returns the identity the folder carries, creating one if it has
	// none. An existing identity is never replaced: it is what every row in the
	// index points at.
	Ensure(root string, at time.Time) (domain.VaultID, error)
}
