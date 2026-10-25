package registry

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ErrNoName is a call that names no vault.
var ErrNoName = errors.New("name the vault, as vault_list gives it")

// be another.
var ErrShowing = errors.New("this is the vault the window is showing")

// ErrUnknown is a name, a path or an identity no vault on the list answers to.
var ErrUnknown = errors.New("this installation holds no such vault")

// Find resolves what the user typed — a name, a path or an identity — into
// a vault. It is shared by every use case that works on one.
type Find struct {
	registry vaultport.VaultRegistry
}

// NewFind is what a name, a path or an identity is resolved against: the list
// this installation keeps.
func NewFind(registry vaultport.VaultRegistry) Find {
	return Find{registry: registry}
}

func (u Find) Execute(nameOrPath string) (domain.Vault, error) {
	if nameOrPath == "" {
		return domain.Vault{}, ErrNoName
	}
	v, found, err := u.registry.FindVault(nameOrPath)
	if err != nil {
		return domain.Vault{}, err
	}
	if !found {
		return domain.Vault{}, fmt.Errorf("%w: no vault called %q", ErrUnknown, nameOrPath)
	}
	return v, nil
}
