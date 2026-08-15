package usecase

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// AddVault turns a folder into a vault this installation knows about.
//
// Two things happen, in this order and no other: the folder is given an
// identity that stays with it, and that identity is remembered here. The order
// matters — a folder recorded in the registry but carrying no identity would be
// unrecognisable the moment it moved.
type AddVault struct {
	Identity port.VaultIdentity
	Registry port.VaultRegistry
	Now      func() time.Time
}

func (u AddVault) Execute(root, name string) (domain.Vault, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return domain.Vault{}, err
	}
	if err := u.Identity.Readable(root); err != nil {
		return domain.Vault{}, err
	}

	now := time.Now
	if u.Now != nil {
		now = u.Now
	}
	id, err := u.Identity.Ensure(root, now())
	if err != nil {
		return domain.Vault{}, fmt.Errorf("give %s an identity: %w", root, err)
	}

	v := domain.Vault{ID: id, Name: name, Path: root}
	if v.Name == "" {
		// The folder name is what the user already calls this collection.
		v.Name = filepath.Base(root)
	}
	if err := u.Registry.Add(v); err != nil {
		return domain.Vault{}, err
	}
	return v, nil
}

// ListVaults reports the vaults this installation knows.
type ListVaults struct {
	Registry port.VaultRegistry
}

func (u ListVaults) Execute() ([]domain.Vault, error) {
	return u.Registry.List()
}

// FindVault resolves what the user typed — a name, a path or an identity — into
// a vault. It is shared by every use case that works on one.
type FindVault struct {
	Registry port.VaultRegistry
}

func (u FindVault) Execute(nameOrPath string) (domain.Vault, error) {
	v, found, err := u.Registry.Find(nameOrPath)
	if err != nil {
		return domain.Vault{}, err
	}
	if !found {
		return domain.Vault{}, fmt.Errorf("no vault called %q", nameOrPath)
	}
	return v, nil
}
