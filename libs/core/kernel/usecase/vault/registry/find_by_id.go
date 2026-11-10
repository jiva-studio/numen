package registry

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// FindByID resolves the identity of a vault into the vault on the list.
type FindByID struct {
	lookup vaultport.VaultLookup
}

func NewFindByID(lookup vaultport.VaultLookup) FindByID {
	return FindByID{lookup: lookup}
}

// Execute returns the vault with the identity, and an error that wraps edit.ErrNotExist for an identity
// no vault on the list has.
func (u FindByID) Execute(id domain.VaultID) (domain.Vault, error) {
	found, isFound, err := u.lookup.FindVault(string(id))
	if err != nil {
		return domain.Vault{}, err
	}
	if !isFound {
		return domain.Vault{}, fmt.Errorf("vault %q: %w", id, edit.ErrNotExist)
	}
	return found, nil
}
