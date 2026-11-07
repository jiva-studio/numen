package vault

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// CheckIdentity rejects a vault whose folder does not carry its identity.
type CheckIdentity struct {
	identity vaultport.VaultIdentity
}

func NewCheckIdentity(identity vaultport.VaultIdentity) CheckIdentity {
	return CheckIdentity{identity: identity}
}

func (u CheckIdentity) Execute(v domain.Vault) error {
	if err := u.identity.CheckReadable(v.Path); err != nil {
		return domain.NewDetailError(fmt.Sprintf("%s cannot be read as a vault: %v", v.Path, err), ErrUnreadable, err)
	}
	carried, found, err := u.identity.GetVaultID(v.Path)
	if err != nil {
		return err
	}
	if !found || carried != v.ID {
		return domain.NewDetailError(fmt.Sprintf("%s is no longer the vault %s", v.Path, v.Name), ErrUnreadable)
	}
	return nil
}
