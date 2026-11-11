package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// EraseConfirmation asks whether the person wants the vault erased.
type EraseConfirmation interface {
	ConfirmErase(v domain.Vault) (bool, error)
}

// Erase is Unregister, and the folder goes to the place this machine keeps what a
// person deleted. The folder is taken away only while it still carries this
// vault's identity.
type Erase struct {
	identity   vaultport.VaultIdentity
	trash      machineport.Trash
	unregister Unregister
}

// NewErase creates an Erase from the vault identity reader, the trash, and the
// Unregister that removes the vault from the list, the same one used for every
// other vault unregistration.
func NewErase(identity vaultport.VaultIdentity, trash machineport.Trash, unregister Unregister) Erase {
	return Erase{identity: identity, trash: trash, unregister: unregister}
}

// EraseResult reports what happened to the folder. Only Erase can tell whether
// the folder was already missing, so the caller receives it from here.
type EraseResult struct {
	Vault     domain.Vault
	IsTrashed bool
}

type EraseByName struct {
	find  VaultFinding
	erase Erase
}

// NewEraseByName creates the use case that resolves the vault with find and erases it with erase.
func NewEraseByName(find VaultFinding, erase Erase) EraseByName {
	return EraseByName{find: find, erase: erase}
}

// ExecuteByName resolves the vault, asks for confirmation when confirm is non-nil, and erases the vault.
func (u EraseByName) ExecuteByName(ctx context.Context, nameOrPath string, confirm EraseConfirmation) (EraseResult, error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return EraseResult{}, err
	}
	if confirm != nil {
		isConfirmed, confirmErr := confirm.ConfirmErase(v)
		if confirmErr != nil {
			return EraseResult{Vault: v}, confirmErr
		}
		if !isConfirmed {
			return EraseResult{Vault: v}, fmt.Errorf("%s was not erased", v.Name)
		}
	}
	res, err := u.erase.Execute(ctx, v)
	res.Vault = v
	return res, err
}

func (u Erase) Execute(ctx context.Context, v domain.Vault) (EraseResult, error) {
	// The last-vault check runs before the folder is moved.
	if err := keepTheLastVault(u.unregister.vaults, v); err != nil {
		return EraseResult{}, err
	}

	err := u.identity.CheckReadable(v.Path)
	if errors.Is(err, edit.ErrNotExist) {
		// Nothing is at the path any more. What is left of the vault is its
		// entry and its rows.
		return EraseResult{}, u.unregister.Execute(ctx, v)
	}
	if err != nil {
		return EraseResult{}, domain.NewDetailError(fmt.Sprintf("%s cannot be read as a vault: %v", v.Path, err), vaultdomain.ErrUnreadable, err)
	}

	carried, carriesOne, err := u.identity.GetVaultID(v.Path)
	if err != nil {
		return EraseResult{}, err
	}
	if !carriesOne || carried != v.ID {
		return EraseResult{}, domain.NewDetailError(fmt.Sprintf("%s is no longer the vault %s; the folder is not moved to the trash", v.Path, v.Name), vaultdomain.ErrUnreadable)
	}

	if err := u.trash.Trash(v.Path); err != nil {
		return EraseResult{}, fmt.Errorf("%s cannot be moved to the trash: %w", v.Path, err)
	}
	return EraseResult{IsTrashed: true}, u.unregister.Execute(ctx, v)
}
