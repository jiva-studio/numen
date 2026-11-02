package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ErrLastVault is the one vault an installation has left. It stays on the list.
var ErrLastVault = errors.New("an installation keeps a vault")

// Unregister takes a vault off the list this installation keeps, and out of the
// index. The folder stays where it is, with the identity it carries, and adding
// it again brings back the same vault.
type Unregister struct {
	registry vaultport.VaultRegistry
	index    vaultport.VaultRepository
}

// NewUnregister is what takes a vault off the list: the list this installation
// keeps, and the index its rows are taken out of.
func NewUnregister(registry vaultport.VaultRegistry, index vaultport.VaultRepository) Unregister {
	return Unregister{registry: registry, index: index}
}

func (u Unregister) Execute(ctx context.Context, v domain.Vault) error {
	if err := keepTheLastVault(u.registry, v); err != nil {
		return err
	}

	// The index goes first: no entry on the list points at rows that are gone.
	// again.
	if err := u.index.Unregister(ctx, v.ID); err != nil {
		return fmt.Errorf("take %s out of the index: %w", v.Name, err)
	}
	return u.registry.Remove(v.ID)
}

// keepTheLastVault rejects a vault that is the only one the installation has left.
func keepTheLastVault(registry vaultport.VaultRegistry, v domain.Vault) error {
	recognized, err := registry.List()
	if err != nil {
		return err
	}
	if len(recognized) == 1 && recognized[0].ID == v.ID {
		return fmt.Errorf("%w: %s is the only vault this installation has", ErrLastVault, v.Name)
	}
	return nil
}

// Remove takes the vault named by an identity off the list, and moves its
// folder to the trash when requested. The current vault is kept.
type Remove struct {
	find       Find
	unregister Unregister
	erase      Erase
}

func NewRemove(find Find, unregister Unregister, erase Erase) Remove {
	return Remove{find: find, unregister: unregister, erase: erase}
}

func (u Remove) Execute(ctx context.Context, id string, showing domain.VaultID, trash bool) error {
	v, err := u.find.Execute(id)
	if err != nil {
		return err
	}
	if v.ID == showing {
		return ErrShowing
	}
	if trash {
		_, err = u.erase.Execute(ctx, v)
		return err
	}
	return u.unregister.Execute(ctx, v)
}
