package sourceport

import (
	"context"
	"io"
)

// ImportedFile is one item the user supplied to this application, by the name
// the source machine gives it.
type ImportedFile struct {
	// Name is what it is called, which is the name it lands under in the vault.
	Name string
	// Handle is what it is opened by. A path on a desktop and a content URI on
	// a phone: whatever produced it is what can read it.
	Handle string
	// IsFolder is whether everything under it is imported with it, and File
	// whether bytes can be read from it. A device, a socket or a link is
	// neither, and stays where it is. A listing names what it holds and reports
	// neither; GetImportedFile supplies both for each file.
	IsFolder bool
	IsFile   bool
}

// ImportedFiles reads what the user supplied to this application from outside
// every vault.
//
// It is the other half of the copy a VaultWriter finishes, and a port for the
// same reason: what arrives is a path on one machine and a content URI on
// another, and callers do not depend on which.
type ImportedFiles interface {
	// GetName is what the machine calls the file at a handle. It reads nothing,
	GetName(handle string) string
	// GetImportedFile returns what is at a handle.
	GetImportedFile(ctx context.Context, handle string) (ImportedFile, error)
	// List is what a folder holds, each by its own handle, without descending.
	List(ctx context.Context, handle string) ([]ImportedFile, error)
	// Open reads one file. The caller that opens it closes it.
	Open(ctx context.Context, handle string) (io.ReadCloser, error)
	// HasHandle reports whether a folder supplied for import contains a vault.
	// Such a folder would be copied into itself, and it is not imported.
	HasHandle(handle, vault string) bool
}
