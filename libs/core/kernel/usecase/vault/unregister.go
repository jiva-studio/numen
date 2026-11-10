package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
)

// ErrLastVault is the one vault an installation has left. It stays on the list.
var ErrLastVault = errors.New("the last vault cannot be removed")

// Unregister takes a vault off the list this installation keeps, and out of the
// index. The folder keeps its position, with the identity it carries, and adding
// it again brings back the same vault.
type Unregister struct {
	vaults  vaultport.VaultList
	remover vaultport.VaultRemover
	index   vaultport.VaultRepository
}

// NewUnregister is what takes a vault off the list: the list this installation
// keeps and its remover, and the index its rows are taken out of.
func NewUnregister(vaults vaultport.VaultList, remover vaultport.VaultRemover, index vaultport.VaultRepository) Unregister {
	return Unregister{vaults: vaults, remover: remover, index: index}
}

func (u Unregister) Execute(ctx context.Context, v domain.Vault) error {
	if err := keepTheLastVault(u.vaults, v); err != nil {
		return err
	}

	// The index goes first: no entry on the list points at rows that are gone.
	// again.
	if err := u.index.Unregister(ctx, v.ID); err != nil {
		return fmt.Errorf("the index cannot remove the vault %s: %w", v.Name, err)
	}
	return u.remover.Remove(v.ID)
}

func keepTheLastVault(vaults vaultport.VaultList, v domain.Vault) error {
	recognized, err := vaults.List()
	if err != nil {
		return err
	}
	if len(recognized) == 1 && recognized[0].ID == v.ID {
		return domain.NewDetailError(fmt.Sprintf("%s is the only vault; the last vault cannot be removed", v.Name), ErrLastVault)
	}
	return nil
}

// Remove takes the vault named by an identity off the list, and moves its
// folder to the trash when requested. The current vault is kept.
type Remove struct {
	find       registry.Find
	unregister Unregister
	erase      Erase
}

func NewRemove(find registry.Find, unregister Unregister, erase Erase) Remove {
	return Remove{find: find, unregister: unregister, erase: erase}
}

func (u Remove) Execute(ctx context.Context, id string, showing domain.VaultID, trash bool) error {
	v, err := u.find.Execute(id)
	if err != nil {
		return err
	}
	if v.ID == showing {
		return domain.NewDetailError(fmt.Sprintf("%s is the vault the window shows", v.Name), registry.ErrShowing)
	}
	if trash {
		_, err = u.erase.Execute(ctx, v)
		return err
	}
	return u.unregister.Execute(ctx, v)
}
