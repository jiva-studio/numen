package machineport

import "errors"

// Trash is where a folder goes when the user is done with it.
type Trash interface {
	// Trash moves a folder to the location where this machine keeps deleted
	// items. It returns ErrNoTrash where this machine has no such location.
	Trash(path string) error
}

// ErrNoTrash is a machine with nowhere to put what is deleted. A folder on a
// mounted volume of its own has a trash of its own, and where that cannot be
// made there is none.
var ErrNoTrash = errors.New("this machine has nowhere to put what is deleted")
