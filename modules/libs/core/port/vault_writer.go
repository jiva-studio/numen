package port

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// ErrChanged is what a write says when the file it was about to replace is no
// longer the one the caller read. Nothing is written: a person editing their
// own note outranks whatever was going to be put on top of it.
var ErrChanged = errors.New("the note changed since it was read")

// ErrOccupied is what a move says when something already sits where a note was
// going. Nothing is moved.
var ErrOccupied = errors.New("a file is already there")

// VaultWriter changes the files of one vault. It is the only way the
// application touches what the person wrote, and it is deliberately small: what
// belongs in a note is the core's business, and where the bytes land is this.
//
// Every path is relative to the vault root, in the form a walk reports it.
// A path that leaves the vault is refused.
type VaultWriter interface {
	// Write puts content at a path, replacing whatever is there and creating
	// the folders above it. It is atomic: a reader sees the note as it was or
	// as it now is, never half of either.
	//
	// Fingerprint, when it is not the zero value, is what the caller believes
	// is on disk. A file that no longer matches it is left alone and
	// ErrChanged is returned: the caller read a note, thought about it, and
	// something else wrote in the meantime.
	//
	// What comes back is the fingerprint of the file this write produced,
	// which is what the caller presents at its next write. It is the zero
	// value when nothing was written.
	Write(ctx context.Context, path string, content []byte, fingerprint domain.FileRef) (domain.FileRef, error)

	// Create puts content where there is nothing, and refuses where there is
	// something. Looking first and writing after is not the same promise: two
	// callers can both look, both find nothing, and the second overwrite the
	// first. Only the filesystem can answer this, and it answers it once.

	Create(ctx context.Context, path string, content []byte) error

	// Move renames a file or a folder, creating the folders above its
	// destination. The bytes do not change, so a note that carried no
	// identifier still carries none afterwards.
	//
	// A destination that is taken is refused: two notes arriving at one path
	// is a question only whoever asked for the move can answer.
	Move(ctx context.Context, from, to string) error

	// MakeFolder puts an empty folder at a path, creating the folders above
	// it. A folder that is already there is the outcome that was asked for; a
	// file there is refused with ErrOccupied.
	MakeFolder(ctx context.Context, path string) error

	// Remove takes a file out of the vault for good. Putting a note in the
	// trash is a move, and is not this.
	Remove(ctx context.Context, path string) error
}

// VaultWriters opens a vault for writing. Which vault a use case works on is
// decided while it runs.
type VaultWriters interface {
	Open(v domain.Vault) (VaultWriter, error)

	// Hold takes one vault's write lock and answers with what gives it back.
	// One write to a vault happens at a time, and a read-modify-write holds
	// the lock from its read to its rename.
	//
	// A context that ends while the lock is waited for is answered with its
	// error, and nothing is held.
	Hold(ctx context.Context, v domain.Vault) (release func(), err error)
}
