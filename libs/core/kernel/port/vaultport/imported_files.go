package vaultport

import (
	"context"
	"io"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

// ImportedFiles reads what the user supplied to this application from outside
// every vault. It is the other half of the copy a VaultWriter finishes, and a
// port because what arrives is a path on one machine and a content URI on
// another.
type ImportedFiles interface {
	// GetName is what the machine calls the file at a handle. It reads nothing,
	GetName(handle string) string
	// GetImportedFile returns what is at a handle.
	GetImportedFile(ctx context.Context, handle string) (vaultdomain.ImportedFile, error)
	// List is what a folder holds, each by its own handle, without descending.
	List(ctx context.Context, handle string) ([]vaultdomain.ImportedFile, error)
	// Open reads one file. The caller that opens it closes it.
	Open(ctx context.Context, handle string) (io.ReadCloser, error)
	// HasHandle reports whether a folder supplied for import contains a vault.
	// Such a folder would be copied into itself, and it is not imported.
	HasHandle(handle, vault string) bool
}
