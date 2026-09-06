package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrNameTaken is a name another vault on the list already has.
var ErrNameTaken = errors.New("another vault is called this")

// Rename gives a vault another name. The name is what a person calls the
// collection and lives in the list this installation keeps; the folder keeps
// the name the filesystem gives it.
type Rename struct {
	Registry port.VaultRegistry
	Index    port.VaultRepository
}

// NewRename is what a vault is called through: the list this installation
// keeps, which is where the name stands, and the index, which holds the row the
// vault's other rows point at.
func NewRename(registry port.VaultRegistry, index port.VaultRepository) Rename {
	return Rename{Registry: registry, Index: index}
}

// Execute answers with the vault under its new name. The name a vault already
// has is not a change and not an error.
func (u Rename) Execute(ctx context.Context, v domain.Vault, name string) (domain.Vault, error) {
	if name == "" {
		return domain.Vault{}, errors.New("a vault is called something")
	}
	if name == v.Name {
		return v, nil
	}

	known, err := u.Registry.All()
	if err != nil {
		return domain.Vault{}, err
	}
	if other, taken := domain.Vaults(known).Called(name, v.ID); taken {
		return domain.Vault{}, fmt.Errorf("%w: the vault at %s is already called %s",
			ErrNameTaken, other.Path, other.Name)
	}

	renamed := v
	renamed.Name = name
	if err := u.Registry.Save(renamed); err != nil {
		return domain.Vault{}, err
	}
	// The index holds no name, so a rename writes nothing to it. This covers a
	// vault renamed before its first scan, which has no row yet to be scanned
	// into.
	if err := u.Index.Register(ctx, renamed.ID); err != nil {
		return domain.Vault{}, fmt.Errorf("register %s in the index: %w", v.Name, err)
	}
	return renamed, nil
}
