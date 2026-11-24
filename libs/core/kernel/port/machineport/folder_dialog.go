package machineport

import "context"

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
	// It returns machine.ErrChoosing when a picker is already open, and
	// machine.ErrNoFolderDialog when this machine cannot show one.
	ChooseFolder(ctx context.Context, title, startingAt string) (string, bool, error)
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
