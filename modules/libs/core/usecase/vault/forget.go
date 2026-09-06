package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrLastVault is the one vault an installation has left. It stays on the list.
var ErrLastVault = errors.New("an installation keeps a vault")

// Forget takes a vault off the list this installation keeps, and out of the
// index. The folder stays where it is, with the identity it carries, and adding
// it again brings back the same vault.
type Forget struct {
	Registry port.VaultRegistry
	Index    port.VaultRepository
}

// NewForget is what takes a vault off the list: the list this installation
// keeps, and the index its rows are taken out of.
func NewForget(registry port.VaultRegistry, index port.VaultRepository) Forget {
	return Forget{Registry: registry, Index: index}
}

func (u Forget) Execute(ctx context.Context, v domain.Vault) error {
	if err := keepTheLastVault(u.Registry, v); err != nil {
		return err
	}

	// The index goes first: no entry on the list points at rows that are gone.
	// Rows a failure here leaves behind belong to a vault the next scan writes
	// again.
	if err := u.Index.Forget(ctx, v.ID); err != nil {
		return fmt.Errorf("take %s out of the index: %w", v.Name, err)
	}
	return u.Registry.Remove(v.ID)
}

// keepTheLastVault refuses a vault that is all the installation has left.
func keepTheLastVault(registry port.VaultRegistry, v domain.Vault) error {
	known, err := registry.All()
	if err != nil {
		return err
	}
	if len(known) == 1 && known[0].ID == v.ID {
		return fmt.Errorf("%w: %s is the only vault this installation has", ErrLastVault, v.Name)
	}
	return nil
}
