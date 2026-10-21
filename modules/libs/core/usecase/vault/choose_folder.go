package vault

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrNoTrash is a machine with nowhere to put what is deleted.
var ErrNoTrash = port.ErrNoTrash

// ChooseFolder puts this machine's folder dialog in front of the person, under
// the title and at the folder it is given.
type ChooseFolder struct {
	dialog port.FolderDialog
}

func NewChooseFolder(dialog port.FolderDialog) ChooseFolder {
	return ChooseFolder{dialog: dialog}
}

// Execute returns the folder the person chose. IsChosen is false when the person
// closed the dialog and chose none.
func (u ChooseFolder) Execute(ctx context.Context, title, startingAt string) (PickedFolder, error) {
	selected, isChosen, err := u.dialog.ChooseFolder(ctx, title, startingAt)
	if err != nil {
		return PickedFolder{}, err
	}
	return PickedFolder{Path: selected, IsChosen: isChosen}, nil
}

// GetPickFolder is the use case that puts the same dialog in front of the person
// under the title a vault is added with.
func (u ChooseFolder) GetPickFolder() PickFolder {
	return NewPickFolder(u.dialog)
}

// IsFolderDialogUnavailable is whether an error says the dialog cannot be put up
// now: one is already open, or this machine has none.
func IsFolderDialogUnavailable(err error) bool {
	return errors.Is(err, port.ErrChoosing) || errors.Is(err, port.ErrNoFolderDialog)
}
