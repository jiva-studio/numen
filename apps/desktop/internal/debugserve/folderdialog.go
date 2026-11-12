//go:build numen_debug

package debugserve

import (
	"context"
	"os"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
)

// FolderFileDialog answers the folder dialog with the folder a test wrote in a file, because a page
// served over HTTP has no window to show the dialog in. A file with no folder answers that the
// person chose none.
type FolderFileDialog struct {
	path string
}

// NewFolderFileDialog returns the dialog that reads the folder it answers with from the file at path.
func NewFolderFileDialog(path string) FolderFileDialog {
	return FolderFileDialog{path: path}
}

// ChooseFolder returns the folder written in the file, and false when the file holds none.
func (d FolderFileDialog) ChooseFolder(_ context.Context, _, _ string) (string, bool, error) {
	text, err := os.ReadFile(d.path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	folder := strings.TrimSpace(string(text))
	return folder, folder != "", nil
}

// ChooseFile answers that this machine shows no file dialog.
func (d FolderFileDialog) ChooseFile(_ context.Context, _, _ string, _ []string) (string, bool, error) {
	return "", false, machine.ErrNoFolderDialog
}
