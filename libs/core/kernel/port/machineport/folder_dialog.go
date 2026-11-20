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
