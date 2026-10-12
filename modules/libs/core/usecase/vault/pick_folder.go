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

// Execute returns the folder the person chose, and false when they chose none.
// A path that is given is returned as it is without asking anybody.
func (u *PickFolder) Execute(ctx context.Context, path string) (string, bool, error) {
	if path != "" {
		return path, true, nil
	}
	if u == nil || u.dialog == nil {
		return "", false, errors.New("there is nobody here to pick a folder: name the one to add")
	}
	return u.dialog.ChooseFolder(ctx, "Choose a folder for a vault", "")
}
