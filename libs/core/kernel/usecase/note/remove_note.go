package note

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// ErrFolder reports an attempt to remove a path that is a folder. Notes are
// removed one by one.
var ErrFolder = errors.New("the path is a folder; pass the paths of the notes to remove")

// noteRemoving is the removal of a note that RemoveNote guards.
type noteRemoving interface {
	Execute(ctx context.Context, v domain.Vault, path string) (RemoveResult, error)
	Destroy(ctx context.Context, v domain.Vault, path string) (RemoveResult, error)
}

// RemoveNote removes one note, to the trash or permanently, and rejects a folder.
type RemoveNote struct {
	remove  noteRemoving
	readers vaultport.VaultReaders
}

func NewRemoveNote(remove noteRemoving, readers vaultport.VaultReaders) RemoveNote {
	return RemoveNote{remove: remove, readers: readers}
}

func (u RemoveNote) Execute(
	ctx context.Context, v domain.Vault, path string, shouldDestroy bool,
) (RemoveResult, error) {
	if u.readers == nil {
		return RemoveResult{}, errNotOpen
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return RemoveResult{}, err
	}
	if _, err := reader.List(ctx, path); err == nil {
		return RemoveResult{}, ErrFolder
	}
	if shouldDestroy {
		return u.remove.Destroy(ctx, v, path)
	}
	return u.remove.Execute(ctx, v, path)
}
