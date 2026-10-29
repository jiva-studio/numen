package picker

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// ErrNoTrash is a machine with nowhere to put what is deleted.
var ErrNoTrash = machineport.ErrNoTrash

// ChooseFolder shows this machine's folder dialog with the given title and
// starting folder.
type ChooseFolder struct {
	dialog machineport.FolderDialog
}

func NewChooseFolder(dialog machineport.FolderDialog) ChooseFolder {
	return ChooseFolder{dialog: dialog}
}

// Execute returns the folder the user chose. IsChosen is false when the user
// closed the dialog without choosing.
func (u ChooseFolder) Execute(ctx context.Context, title, startingAt string) (PickedFolder, error) {
	selected, isChosen, err := u.dialog.ChooseFolder(ctx, title, startingAt)
	if err != nil {
		return PickedFolder{}, err
	}
	return PickedFolder{Path: selected, IsChosen: isChosen}, nil
}

// GetPickFolder returns the use case that shows the same dialog with the title
// used when adding a vault.
func (u ChooseFolder) GetPickFolder() PickFolder {
	return NewPickFolder(u.dialog)
}

// IsFolderDialogUnavailable reports whether an error means the dialog cannot be
// shown: one is already open, or this machine has none.
func IsFolderDialogUnavailable(err error) bool {
	return errors.Is(err, machineport.ErrChoosing) || errors.Is(err, machineport.ErrNoFolderDialog)
}
