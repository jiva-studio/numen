package vault

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Erase is Forget, and the folder goes to the place this machine keeps what a
// person deleted. The folder is taken away only while it still carries this
// vault's identity.
type Erase struct {
	Identity port.VaultIdentity
	Trash    port.Trash
	Forget   Forget
}

func (u Erase) Execute(ctx context.Context, v domain.Vault) error {
	// Asked before the folder is moved.
	if err := keepTheLastVault(u.Forget.Registry, v); err != nil {
		return err
	}

	err := u.Identity.Readable(v.Path)
	if errors.Is(err, fs.ErrNotExist) {
		// Nothing is at the path any more. What is left of the vault is its
		// entry and its rows.
		return u.Forget.Execute(ctx, v)
	}
	if err != nil {
		return err
	}

	carried, carriesOne, err := u.Identity.Of(v.Path)
	if err != nil {
		return err
	}
	if !carriesOne || carried != v.ID {
		return fmt.Errorf("%s no longer carries the identity of the vault %s, so it stays where it is", v.Path, v.Name)
	}

	if err := u.Trash.Trash(v.Path); err != nil {
		return fmt.Errorf("move %s to the trash: %w", v.Path, err)
	}
	return u.Forget.Execute(ctx, v)
}
