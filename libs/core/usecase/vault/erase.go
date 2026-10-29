package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

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
	// IsTrashed is whether the folder went to the trash. A vault whose folder had
	// already gone leaves nothing to move, and only its entry and its rows go.
	IsTrashed bool
}

func (u Erase) Execute(ctx context.Context, v domain.Vault) (EraseResult, error) {
	// The last-vault check runs before the folder is moved.
	if err := keepTheLastVault(u.unregister.registry, v); err != nil {
		return EraseResult{}, err
	}

	err := u.identity.CheckReadable(v.Path)
	if errors.Is(err, vaultport.ErrNotExist) {
		// Nothing is at the path any more. What is left of the vault is its
		// entry and its rows.
		return EraseResult{}, u.unregister.Execute(ctx, v)
	}
	if err != nil {
		return EraseResult{}, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	carried, carriesOne, err := u.identity.GetVaultID(v.Path)
	if err != nil {
		return EraseResult{}, err
	}
	if !carriesOne || carried != v.ID {
		return EraseResult{}, fmt.Errorf(
			"%w: %s no longer carries the identity of the vault %s, so it stays where it is",
			ErrUnreadable, v.Path, v.Name)
	}

	if err := u.trash.Trash(v.Path); err != nil {
		return EraseResult{}, fmt.Errorf("move %s to the trash: %w", v.Path, err)
	}
	return EraseResult{IsTrashed: true}, u.unregister.Execute(ctx, v)
}
