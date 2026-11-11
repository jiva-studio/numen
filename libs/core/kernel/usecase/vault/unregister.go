package vault

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Unregister takes a vault off the list this installation keeps, and out of the
// index. The folder keeps its position, with the identity it carries, and adding
// it again brings back the same vault.
type Unregister struct {
	find    VaultFinding
	vaults  vaultport.VaultList
	remover vaultport.VaultRemover
	index   vaultport.VaultRepository
}

// NewUnregister is what takes a vault off the list: the list this installation
// keeps and its remover, and the index its rows are taken out of.
func NewUnregister(vaults vaultport.VaultList, remover vaultport.VaultRemover, index vaultport.VaultRepository) Unregister {
	return Unregister{vaults: vaults, remover: remover, index: index}
}

// WithFind sets the lookup that resolves a name, path or identity into a vault.
func (u Unregister) WithFind(find VaultFinding) Unregister {
	u.find = find
	return u
}

// ExecuteByName resolves the vault and takes it off the list and out of the index.
func (u Unregister) ExecuteByName(ctx context.Context, nameOrPath string) (domain.Vault, error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return domain.Vault{}, err
	}
	if err := u.Execute(ctx, v); err != nil {
		return domain.Vault{}, err
	}
	return v, nil
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
		return domain.NewDetailError(fmt.Sprintf("%s is the only vault; the last vault cannot be removed", v.Name), vaultdomain.ErrLastVault)
	}
	return nil
}

// VaultFinding resolves what the user typed, a name, a path or an identity, into a vault on the list.
type VaultFinding func(nameOrPath string) (domain.Vault, error)

// Remove takes the vault named by an identity off the list, and moves its
// folder to the trash when requested. The current vault is kept.
type Remove struct {
	find       VaultFinding
	unregister Unregister
	erase      Erase
}

func NewRemove(find VaultFinding, unregister Unregister, erase Erase) Remove {
	return Remove{find: find, unregister: unregister, erase: erase}
}

func (u Remove) Execute(ctx context.Context, id string, showing domain.VaultID, trash bool) error {
	v, err := u.find(id)
	if err != nil {
		return err
	}
	if v.ID == showing {
		return domain.NewDetailError(fmt.Sprintf("%s is the vault the window shows", v.Name), vaultdomain.ErrShowing)
	}
	if trash {
		_, err = u.erase.Execute(ctx, v)
		return err
	}
	return u.unregister.Execute(ctx, v)
}
