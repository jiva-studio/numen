package vaultport

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// VaultReader is one vault as seen from outside: something that can be walked
// and read. The filesystem is one implementation; a fixture in memory is
// another.
type VaultReader interface {
	// WalkFiles reports every source the vault holds, in unspecified order, each
	// saying which kind it is. The service folder is not reported, and neither
	// is a file of no kind the application reads.
	WalkFiles(ctx context.Context, fn func(domain.FileStat) error) error
	// List reports the entries of one folder, without descending: the files of
	// every kind and the folders under it. The empty path is the vault root.
	//
	// What a walk leaves out is left out here as well. Folders come first,
	// then files, each by name compared without regard to case, and that is
	// the order to draw them in.
	//
	// vault.ErrOutside for a path that leaves the vault, fs.ErrNotExist for a folder
	// that is not there and for a path that holds a file.
	List(ctx context.Context, folder string) ([]domain.Entry, error)
	// Read returns the bytes of one file, addressed by a path a walk reported.
	Read(ctx context.Context, path string) ([]byte, error)
	// Open returns a file for partial reads. A recording is hours long and is
	// played from the middle, so reading the whole file into memory for every
	// listener is not acceptable.
	//
	// The errors are the same as Read. The caller closes the file.
	Open(ctx context.Context, path string) (io.ReadSeekCloser, error)
	// StatFile returns what a walk reports about one path: its kind, size and
	// modification time. It reads no bytes, so it is the call for a file the
	// caller does not want to open.
	//
	// It returns fs.ErrNotExist when there is nothing at the path, and
	// ErrNotANote when the file there is one the vault ignores.
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
}

// VaultReaders opens a reader for a given vault. Which vault a use case works
// on is decided while it runs.
type VaultReaders interface {
	Open(v domain.Vault) (VaultReader, error)
}
