package trash

import (
	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// ErrNoTrash is the core's sentinel, under the name this package answers by.
var ErrNoTrash = machineport.ErrNoTrash

// Trash is this machine's trash.
type Trash struct{}

// New is this machine's trash.
func New() Trash { return Trash{} }

// Trash moves the folder to this machine's trash location. The path is absolute
// and names an existing directory, which is moved as given: nothing on the way
// to it is resolved here.
func (Trash) Trash(path string) error { return moveToTrash(path) }
