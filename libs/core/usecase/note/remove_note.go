package note

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ErrFolder is what removing a path that names a folder says. Notes are removed
// by naming each of them.
var ErrFolder = errors.New("this is a folder, and this removes notes: name the notes to remove")

// RemoveNote removes one note, to the trash or outright, and refuses a folder.
type RemoveNote struct {
	remove  Remove
	readers vaultport.VaultReaders
}

func NewRemoveNote(remove Remove, readers vaultport.VaultReaders) RemoveNote {
	return RemoveNote{remove: remove, readers: readers}
}

// Execute removes the note at path, or destroys it when shouldDestroy is set.
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
