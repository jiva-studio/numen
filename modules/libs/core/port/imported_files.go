package port

import (
	"context"
	"io"
)

// ImportedFile is one thing a person handed this application, by the name the
// machine it came from gives it.
type ImportedFile struct {
	// Name is what it is called, which is the name it lands under in the vault.
	Name string
	// Handle is what it is opened by. A path on a desktop and a content URI on
	// a phone: whatever produced it is what can read it.
	Handle string
	// IsFolder is whether everything under it comes in with it, and File whether
	// bytes can be read from it. A device, a socket or a link is neither, and
	// stays where it is. A listing names what it holds and says neither; what
	// one of them is comes from Stat.
	IsFolder bool
	IsFile   bool
}

// ImportedFiles reads what a person handed this application from outside every
// vault.
//
// It is the other half of the copy a VaultWriter finishes, and a port for the
// same reason: what arrives is a path on one machine and a content URI on
// another, and nothing above this knows which.
type ImportedFiles interface {
	// GetName is what the machine calls the file at a handle. It reads nothing,
	// so a handle that cannot be opened still has a name to be refused under.
	GetName(handle string) string
	// Stat says what is at a handle.
	Stat(ctx context.Context, handle string) (ImportedFile, error)
	// List is what a folder holds, each by its own handle, without descending.
	List(ctx context.Context, handle string) ([]ImportedFile, error)
	// Open reads one file. Whoever opens it closes it.
	Open(ctx context.Context, handle string) (io.ReadCloser, error)
	// Contains reports whether a folder handed in is one a vault sits under.
	// Such a folder would be copied into itself, and it does not come in.
	Contains(handle, vault string) bool
}
