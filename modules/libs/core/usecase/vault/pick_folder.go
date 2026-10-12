package vault

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// PickFolder puts this machine's folder dialog in front of the person.
type PickFolder struct {
	dialog port.FolderDialog
}

func NewPickFolder(dialog port.FolderDialog) *PickFolder {
	return &PickFolder{dialog: dialog}
}

// PickedFolder is the folder to add. IsChosen is false when the person closed
// the dialog and chose none.
type PickedFolder struct {
	Path     string
	IsChosen bool
}

// Execute returns the folder the person chose. A path that is given is returned
// as it is without asking anybody.
func (u *PickFolder) Execute(ctx context.Context, path string) (PickedFolder, error) {
	if path != "" {
		return PickedFolder{Path: path, IsChosen: true}, nil
	}
	if u == nil || u.dialog == nil {
		return PickedFolder{}, errors.New("there is nobody here to pick a folder: name the one to add")
	}
	chosen, isChosen, err := u.dialog.ChooseFolder(ctx, "Choose a folder for a vault", "")
	if err != nil {
		return PickedFolder{}, err
	}
	return PickedFolder{Path: chosen, IsChosen: isChosen}, nil
}
