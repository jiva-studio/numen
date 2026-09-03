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
	// The list is what the name is on, and the index holds a copy of it.
	if err := u.Registry.Save(renamed); err != nil {
		return domain.Vault{}, err
	}
	if err := u.Index.Save(ctx, renamed); err != nil {
		return domain.Vault{}, fmt.Errorf("rename %s in the index: %w", v.Name, err)
	}
	return renamed, nil
}
