// Package machine holds the timing functions, the sentinel errors and the worker sizing of the machine the program runs on.
package machine

import "errors"

var (
	// ErrNoTrash is a machine with no trash. A folder on a
	// mounted volume of its own has a trash of its own, and where that cannot be
	// made there is none.
	ErrNoTrash = errors.New("this machine has no trash")

	// ErrChoosing is returned when a picker is requested while one is already open.
	// At most one picker is open at a time.
	ErrChoosing = errors.New("a folder picker is already open")

	// ErrNoFolderDialog is returned when a machine cannot show a folder picker. A folder is
	// then supplied to the application another way, and a vault is added by its path.
	ErrNoFolderDialog = errors.New("this machine cannot show a folder picker")
)
