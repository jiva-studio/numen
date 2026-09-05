package vault

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Erase is Forget, and the folder goes to the place this machine keeps what a
// person deleted. The folder is taken away only while it still carries this
// vault's identity.
type Erase struct {
	Identity port.VaultIdentity
	Trash    port.Trash
	Forget   Forget
}

// NewErase is what takes a vault away: what reads the identity the folder
// carries, where this machine keeps what a person deleted, and the Forget it is
// taken off the list through — the same one everything else forgets a vault by.
func NewErase(identity port.VaultIdentity, trash port.Trash, forget Forget) Erase {
	return Erase{Identity: identity, Trash: trash, Forget: forget}
}

// EraseResult says what became of the folder. Whoever asked for the erasure is
// owed the answer, and only this knows it: a folder that was already gone is
// looked for here and nowhere else.
type EraseResult struct {
	// Trashed is whether the folder went to the trash. A vault whose folder had
	// already gone leaves nothing to move, and only its entry and its rows go.
	Trashed bool
}

func (u Erase) Execute(ctx context.Context, v domain.Vault) (EraseResult, error) {
	// Asked before the folder is moved.
	if err := keepTheLastVault(u.Forget.Registry, v); err != nil {
		return EraseResult{}, err
	}

	err := u.Identity.Readable(v.Path)
	if errors.Is(err, fs.ErrNotExist) {
		// Nothing is at the path any more. What is left of the vault is its
		// entry and its rows.
		return EraseResult{}, u.Forget.Execute(ctx, v)
	}
	if err != nil {
		return EraseResult{}, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	carried, carriesOne, err := u.Identity.Of(v.Path)
	if err != nil {
		return EraseResult{}, err
	}
	if !carriesOne || carried != v.ID {
		return EraseResult{}, fmt.Errorf(
			"%w: %s no longer carries the identity of the vault %s, so it stays where it is",
			ErrUnreadable, v.Path, v.Name)
	}

	if err := u.Trash.Trash(v.Path); err != nil {
		return EraseResult{}, fmt.Errorf("move %s to the trash: %w", v.Path, err)
	}
	return EraseResult{Trashed: true}, u.Forget.Execute(ctx, v)
}
