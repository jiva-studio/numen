package catalog

import (
	"context"
	"errors"
	"io"
	"time"
)

// Source supplies the entries of one place plugins are found in. A source that reads part of its
// place returns the entries it read together with an error that states what it could not read; the
// Service lists the entries and reports the error.
type Source interface {
	// Name identifies the source. It is the same for every call.
	Name() string
	// List returns every entry of the source.
	List(ctx context.Context) ([]Entry, error)
	// Get returns the entry of the plugin, or an error that is ErrNotFound when the source lists none.
	Get(ctx context.Context, id string) (Entry, error)
}

// Snapshot is the entries of a source as they were read at one time.
type Snapshot struct {
	Entries []Entry
	At      time.Time
}

// ErrEmptyCache is a cache that holds no snapshot.
var ErrEmptyCache = errors.New("the catalog cache holds no snapshot")

// Cache keeps the last snapshot of a source.
type Cache interface {
	// Read returns the snapshot, or an error that is ErrEmptyCache when there is none.
	Read(ctx context.Context) (Snapshot, error)
	// Write replaces the snapshot.
	Write(ctx context.Context, snapshot Snapshot) error
}

// Installer puts a plugin of an entry on disk and takes it off. It is declared for the adapters of
// the sources that offer plugins to install.
type Installer interface {
	// Install fetches the plugin of the entry, checks it and unpacks it into the install folder.
	Install(ctx context.Context, entry Entry) error
	// Uninstall removes the folder of the plugin from the install folder.
	Uninstall(ctx context.Context, id string) error
}

// The failures of a PackageCache.
var (
	// ErrPackageNotCached is a package that the cache holds under no such checksum.
	ErrPackageNotCached = errors.New("package is not cached")
	// ErrChecksumMismatch is content whose SHA-256 differs from the checksum it is stored under.
	ErrChecksumMismatch = errors.New("package content does not match its checksum")
	// ErrInvalidChecksum is a checksum that is not 64 lower-case hexadecimal digits.
	ErrInvalidChecksum = errors.New("checksum is not 64 lower-case hex digits")
)

// PackageCache holds downloaded plugin packages by the SHA-256 of their content. Every application
// of the user shares one, and what it holds is checked on every use, since the folder is writable by
// any of them.
type PackageCache interface {
	// Put stores the content under the checksum. Content whose hash differs is refused with
	// ErrChecksumMismatch and leaves nothing behind.
	Put(ctx context.Context, checksum string, content io.Reader) error
	// Open returns the content stored under the checksum after checking its hash. A package that is
	// absent is ErrPackageNotCached, and one that fails the check is removed and is ErrChecksumMismatch.
	Open(ctx context.Context, checksum string) (io.ReadCloser, error)
	// Remove deletes the package. A package that is absent is not an error.
	Remove(ctx context.Context, checksum string) error
}
