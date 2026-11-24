package picker

import (
	"context"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/atomicfile"
)

// TextFile is a text the person keeps as a file: the dialog is titled Title, proposes Name, and offers files of
// Extension, with its dot, or every file when it is empty.
type TextFile struct {
	Title     string
	Name      string
	Extension string
	Content   string
}

// SavedFile is the file a text was written to. IsSaved is false when the user closed the dialog without
// choosing, and nothing was written.
type SavedFile struct {
	Path    string
	IsSaved bool
}

// SaveFileDialog lets the user choose the file to save to on this machine.
type SaveFileDialog interface {
	// ChooseSaveFile shows the machine's native save dialog. It returns the file the user
	// chose, and false when none was chosen.
	//
	// The dialog proposes name and shows the files whose extension is one of accept, or every
	// file when accept is empty.
	//
	// It returns machine.ErrChoosing when a dialog is already open, and machine.ErrNoFolderDialog
	// when this machine cannot show one.
	ChooseSaveFile(ctx context.Context, title, name string, accept []string) (string, bool, error)
}

// SaveText asks the person for a file in this machine's save dialog and writes a text there.
type SaveText struct {
	dialog SaveFileDialog
}

func NewSaveText(dialog SaveFileDialog) SaveText {
	return SaveText{dialog: dialog}
}

// Execute writes the text to the file the user chose. Only the user can read the file.
func (u SaveText) Execute(ctx context.Context, file TextFile) (SavedFile, error) {
	var accept []string
	if file.Extension != "" {
		accept = []string{file.Extension}
	}
	path, isChosen, err := u.dialog.ChooseSaveFile(ctx, file.Title, file.Name, accept)
	if err != nil || !isChosen {
		return SavedFile{}, err
	}
	if err := atomicfile.WriteFile(path, strings.NewReader(file.Content), 0o600); err != nil {
		return SavedFile{}, err
	}
	return SavedFile{Path: path, IsSaved: true}, nil
}
