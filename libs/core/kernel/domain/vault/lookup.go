package vault

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
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

// FindByName resolves what a person typed, a name, a path or an identity, with the lookup of the vault list.
func FindByName(name string, lookup func(nameOrPath string) (domain.Vault, bool, error)) (domain.Vault, error) {
	if name == "" {
		return domain.Vault{}, ErrNoName
	}
	found, isFound, err := lookup(name)
	if err != nil {
		return domain.Vault{}, &LookupError{Name: name, Err: err}
	}
	if !isFound {
		return domain.Vault{}, &UnknownError{Name: name}
	}
	return found, nil
}
