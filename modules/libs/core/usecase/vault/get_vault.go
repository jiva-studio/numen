package vault

import (
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// ErrNoName is a call that names no vault.
var ErrNoName = errors.New("name the vault, as vault_list gives it")

// ErrShowing is the vault the window is showing, where the vault asked for must
// be another.
var ErrShowing = errors.New("this is the vault the window is showing")

// GetVault finds the vault a name, a folder or an identity reaches on the
// list, and refuses the one being shown where that is asked.
type GetVault struct {
	find Find
}

func NewGetVault(find Find) *GetVault {
	return &GetVault{find: find}
}

// Execute returns the vault. A showing of the empty identity refuses nothing,
// and otherwise the vault of that identity is returned with ErrShowing.
func (u *GetVault) Execute(nameOrPath string, showing domain.VaultID) (domain.Vault, error) {
	if nameOrPath == "" {
		return domain.Vault{}, ErrNoName
	}
	v, err := u.find.Execute(nameOrPath)
	if err != nil {
		return domain.Vault{}, err
	}
	if showing != "" && v.ID == showing {
		return v, ErrShowing
	}
	return v, nil
}
