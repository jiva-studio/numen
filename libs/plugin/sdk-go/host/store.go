package host

import "context"

// StoreFile is a file of the plugin's folder: its name, and how many bytes it holds.
type StoreFile struct {
	Name string
	Size int64
}

// StoreReader reads the files of the plugin's own folder in a vault's store, `.numen/<plugin id>/`.
// A name is a file directly in the folder; one containing a slash or beginning with a dot yields
// ErrOutside. A missing file yields ErrNotExist, and a call while the host shows no vault yields
// ErrNoVault. A plugin that works another vault than the shown one calls With on StoreClient and
// keeps the narrow interface of the calls it needs on the result.
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
