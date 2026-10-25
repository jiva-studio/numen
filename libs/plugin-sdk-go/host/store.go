package host

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// StoreFile is a file of the plugin's folder: its name, and how many bytes it holds.
type StoreFile struct {
	Name string
	Size int64
}

// Store is the plugin's own folder in a vault's store, `.numen/<plugin id>/`. Every plugin has one,
// and reaches no other. A plugin works it through StoreClient; tests work it through hosttest.
//
// A name is a file directly in the folder; one that holds a slash or begins with a dot gets
// ErrOutside. A file that is not there gets ErrNotExist, and a call while the host shows no vault
// gets ErrNoVault.
type Store interface {
	// With is the folder in the vault of the identity. A call on it returns ErrUnknownVault when the
	// installation knows no such vault.
	With(id domain.VaultID) Store

	// Read returns the whole file.
	Read(ctx context.Context, name string) ([]byte, error)
	// Append adds to the end of the file, making it when it is not there. The content lands whole or
	// not at all.
	Append(ctx context.Context, name string, content []byte) error
	// Write replaces the whole file, making it when it is not there. The content lands whole or not
	// at all.
	Write(ctx context.Context, name string, content []byte) error
	// Delete removes the file. A file that is not there is removed.
	Delete(ctx context.Context, name string) error
	// List returns the files of the folder, sorted by name. A folder that is not there holds none.
	List(ctx context.Context) ([]StoreFile, error)
}
