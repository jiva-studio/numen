package picker

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// PickedFolder is the folder to add. IsChosen is false when the user closed the
// dialog without choosing.
type PickedFolder struct {
	Path     string
	IsChosen bool
}

// PickFolder opens this machine's folder dialog.
type PickFolder struct {
	dialog machineport.FolderDialog
}

func NewPickFolder(dialog machineport.FolderDialog) PickFolder {
	return PickFolder{dialog: dialog}
}

// Execute returns the folder the user chose.
func (u PickFolder) Execute(ctx context.Context) (PickedFolder, error) {
	selected, isChosen, err := u.dialog.ChooseFolder(ctx, "Choose a folder for a vault", "")
	if err != nil {
		return PickedFolder{}, err
	}
	return PickedFolder{Path: selected, IsChosen: isChosen}, nil
}
