// Package vault holds the sentinel errors of the vault paths and files.
package vault

import "errors"

var (
	// ErrOutside is what a path that does not stay in the vault gets. It is not
	// "no note": a caller asking for something outside has made a mistake worth
	// being notified of, and "no such note" would hide it.
	ErrOutside = errors.New("the path is outside the vault")

	// ErrClaimed is what a name another caller holds gets. Work whose whole output
	// is one file reads it as that work already being under way.
	ErrClaimed = errors.New("the name is claimed by another caller")

	// ErrLockedByAnother is what a name gets when a program outside this one has the
	// file open and the system will not open it twice. A synchroniser and a backup
	// reader each hold a file for moments at a time, so it is a state to be restored
	// to and not a failure of the store.
	ErrLockedByAnother = errors.New("the file is held by another program")

	// ErrAlreadyExists is the move error returned when something already exists
	// where a note was to go.
	ErrAlreadyExists = errors.New("the file already exists")

	// ErrUnreadable is returned for a path that is not a folder, or a folder this
	// application cannot read as the requested vault.
	ErrUnreadable = errors.New("the folder cannot be read as a vault")

	// ErrCopy is one identity carried by two folders that are both there. One index
	// cannot hold both.
	ErrCopy = errors.New("the folder is a copy of a vault on the list")

	// ErrNameTaken is a name another vault on the list already has.
	ErrNameTaken = errors.New("another vault has this name")

	// ErrLastVault is the one vault an installation has left. It stays on the list.
	ErrLastVault = errors.New("the last vault cannot be removed")

	// ErrShowing is returned when the vault to remove is the one the window shows.
	ErrShowing = errors.New("the vault is open in the window")
)
