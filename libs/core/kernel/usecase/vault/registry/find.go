package registry

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// ErrNoName is a call that names no vault.
var ErrNoName = errors.New("the call names no vault")

// ErrUnknown is a name, a path or an identity no vault on the list answers to.
var ErrUnknown = errors.New("the vault is not on the list")

// UnknownError is ErrUnknown for the name, path or identity that was looked up.
type UnknownError struct {
	Name string
}

func (e *UnknownError) Error() string {
	return fmt.Sprintf("no vault on the list has the name, path or identity %q", e.Name)
}

func (e *UnknownError) Unwrap() error { return ErrUnknown }

// LookupError is a failure of the vault list when resolving a name, path or identity.
type LookupError struct {
	Name string
	Err  error
}

func (e *LookupError) Error() string { return e.Err.Error() }

func (e *LookupError) Unwrap() error { return e.Err }

// Find resolves what the user typed — a name, a path or an identity — into
// a vault. It is shared by every use case that works on one.
type Find struct {
	lookup vaultport.VaultLookup
}

// NewFind is what a name, a path or an identity is resolved against: the list
// this installation keeps.
func NewFind(lookup vaultport.VaultLookup) Find {
	return Find{lookup: lookup}
}

func (u Find) Execute(nameOrPath string) (domain.Vault, error) {
	if nameOrPath == "" {
		return domain.Vault{}, ErrNoName
	}
	v, found, err := u.lookup.FindVault(nameOrPath)
	if err != nil {
		return domain.Vault{}, &LookupError{Name: nameOrPath, Err: err}
	}
	if !found {
		return domain.Vault{}, &UnknownError{Name: nameOrPath}
	}
	return v, nil
}
