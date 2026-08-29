package trash

import "github.com/jiva-studio/numen/modules/libs/core/port"

// ErrNoTrash is the core's sentinel, under the name this package answers by.
var ErrNoTrash = port.ErrNoTrash

// Trash is this machine's trash.
type Trash struct{}

var _ port.Trash = Trash{}

// New is this machine's trash.
func New() Trash { return Trash{} }

// Trash moves the folder to where this machine keeps what a person deleted. The
// path is absolute, names a directory that is there, and is what goes: nothing
// on the way to it is resolved here.
func (Trash) Trash(path string) error { return send(path) }
