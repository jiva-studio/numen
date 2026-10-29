package machineport

import (
	"context"
	"errors"
)

// ErrChoosing is returned when a picker is requested while one is already open.
// At most one picker is open at a time.
var ErrChoosing = errors.New("a folder picker is already open")

// ErrNoFolderDialog is returned when a machine cannot show a folder picker. A folder is
// then supplied to the application another way, and a vault is added by its path.
var ErrNoFolderDialog = errors.New("this machine cannot put a folder picker up")

// FolderDialog lets the user pick a folder on this machine.
//
// Only an application with a window can implement it; without a window,
// no dialog is shown.
type FolderDialog interface {
	// ChooseFolder shows the machine's native picker. It returns the chosen
	// folder, and false when none was chosen.
	//
	// The picker has the title title and opens at startingAt; an empty value
	// selects the machine default.
	//
	// It returns ErrChoosing when a picker is already open, and
	// ErrNoFolderDialog when this machine cannot show one.
	ChooseFolder(ctx context.Context, title, startingAt string) (string, bool, error)
}

// PathDialog lets the user pick a file or a folder on this machine.
type PathDialog interface {
	FolderDialog
	// ChooseFile shows the machine's native file picker, listing files whose
	// extension is one of accept, or every file when accept is empty. It behaves like ChooseFolder.
	ChooseFile(ctx context.Context, title, startingAt string, accept []string) (string, bool, error)
}
