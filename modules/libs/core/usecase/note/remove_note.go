package note

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrFolder is what removing a path that names a folder says. Notes are removed
// by naming each of them.
var ErrFolder = errors.New("this is a folder, and this removes notes: name the notes to remove")

// RemoveNote removes one note, to the trash or outright, and refuses a folder.
type RemoveNote struct {
	remove  Remove
	readers port.VaultReaders
}

func NewRemoveNote(remove Remove, readers port.VaultReaders) RemoveNote {
	return RemoveNote{remove: remove, readers: readers}
}

// Execute removes the note at path, or destroys it when shouldDestroy is set.
func (u RemoveNote) Execute(
	ctx context.Context, v domain.Vault, path string, shouldDestroy bool,
) (RemoveResult, error) {
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

// MoveInto files notes under a folder, keeping their names.
type MoveInto struct {
	move Move
}

func NewMoveInto(move Move) MoveInto {
	return MoveInto{move: move}
}

// Execute files the note at path under the folder, which is empty for the root.
func (u MoveInto) Execute(ctx context.Context, v domain.Vault, path, folder string) (MoveResult, error) {
	return u.move.Execute(ctx, v, path, GetPathUnder(folder, path))
}
