package host

import (
	"context"
)

// StoreFile is a file of the plugin's folder: its name, and how many bytes it holds.
type StoreFile struct {
	Name string
	Size int64
}

// Store is the plugin's own folder in a vault's store, `.numen/<plugin id>/`.
// Every plugin has one and cannot access any other. A plugin uses it through
// StoreClient; tests use hosttest.
//
// A name is a file directly in the folder; a name that contains a slash or
// begins with a dot yields ErrOutside. A missing file yields ErrNotExist, and a
// call while the host shows no vault yields ErrNoVault.
type Store interface {
	// With returns the folder in the vault with the given identity. A call on it
	// returns ErrUnknownVault when the installation has no such vault.
	With(id VaultID) Store
	StoreReader
	StoreWriter
}

// StoreReader reads the files of a store folder.
type StoreReader interface {
	// Read returns the whole file.
	Read(ctx context.Context, name string) ([]byte, error)
	// List returns the files of the folder, sorted by name. A folder that is not there holds none.
	List(ctx context.Context) ([]StoreFile, error)
}

// StoreWriter changes the files of a store folder.
type StoreWriter interface {
	// Append adds to the end of the file, making it when it is not there. The content lands whole or
	// not at all.
	Append(ctx context.Context, name string, content []byte) error
	// Write replaces the whole file, making it when it is not there. The content lands whole or not
	// at all.
	Write(ctx context.Context, name string, content []byte) error
	// Delete removes the file. A file that is not there is removed.
	Delete(ctx context.Context, name string) error
	// Rename moves the file to another name of the folder, replacing what the other held. The file
	// lands under the new name whole.
	Rename(ctx context.Context, from, to string) error
}
