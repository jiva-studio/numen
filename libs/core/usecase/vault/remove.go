package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

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
