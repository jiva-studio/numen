package vault

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// CheckIdentity rejects a vault whose folder no longer carries its identity.
type CheckIdentity struct {
	identity vaultport.VaultIdentity
}

func NewCheckIdentity(identity vaultport.VaultIdentity) CheckIdentity {
	return CheckIdentity{identity: identity}
}

func (u CheckIdentity) Execute(v domain.Vault) error {
	if err := u.identity.CheckReadable(v.Path); err != nil {
		return fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	carried, found, err := u.identity.GetVaultID(v.Path)
	if err != nil {
		return err
	}
	if !found || carried != v.ID {
		return fmt.Errorf("%w: %s is no longer the vault %s", ErrUnreadable, v.Path, v.Name)
	}
	return nil
}
