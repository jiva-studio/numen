package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ErrNameTaken is a name another vault on the list already has.
var ErrNameTaken = errors.New("another vault has this name")

// Rename changes a vault's name. The name is the collection's display name and
// lives in the list this installation keeps; the folder keeps the name assigned
// by the filesystem.
type Rename struct {
	registry vaultport.VaultRegistry
	index    vaultport.VaultRepository
}

// NewRename creates a Rename from the list this installation keeps, which stores
// the name, and the index, which holds the row the vault's other rows point at.
func NewRename(registry vaultport.VaultRegistry, index vaultport.VaultRepository) Rename {
	return Rename{registry: registry, index: index}
}

// Execute answers with the vault under its new name. The name a vault already
// has is not a change and not an error.
func (u Rename) Execute(ctx context.Context, v domain.Vault, name string) (domain.Vault, error) {
	if name == "" {
		return domain.Vault{}, errors.New("the new name is empty; a vault needs a name")
	}
	if name == v.Name {
		return v, nil
	}

	recognized, err := u.registry.List()
	if err != nil {
		return domain.Vault{}, err
	}
	if other, exists := domain.Vaults(recognized).FindByName(name, v.ID); exists {
		return domain.Vault{}, domain.NewDetailError(fmt.Sprintf("the vault at %s already has the name %s", other.Path, other.Name), ErrNameTaken)
	}

	renamedVault := v
	renamedVault.Name = name
	if err := u.registry.Save(renamedVault); err != nil {
		return domain.Vault{}, err
	}
	// The index holds no name, so a rename writes nothing to it. This covers a
	// vault renamed before its first scan, which has no row yet to be scanned
	// into.
	if err := u.index.Register(ctx, renamedVault.ID); err != nil {
		return domain.Vault{}, fmt.Errorf("the index cannot record the vault %s: %w", v.Name, err)
	}
	return renamedVault, nil
}
