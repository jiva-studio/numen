package textport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// VersionReader reads the text versions of one vault's files. A version is
// named by its path and its id.
type VersionReader interface {
	// GetVersions is the versions of one file, in the order they were first saved.
	GetVersions(ctx context.Context, path string) ([]domain.TextVersion, error)
	// GetVersionsUnder is the versions of a file, or of everything under a folder.
	GetVersionsUnder(ctx context.Context, path string) ([]domain.TextVersion, error)
}

// VersionWriter records and removes the text versions of one vault's files.
type VersionWriter interface {
	// SaveVersion records a version, replacing one of the same path and id in its place.
	SaveVersion(ctx context.Context, version domain.TextVersion) error
	// RemoveVersion removes a version. A version that is not there is removed.
	RemoveVersion(ctx context.Context, path string, id domain.TextVersionID) error
	// RemovePluginVersions removes every version a plugin made, and returns them.
	RemovePluginVersions(ctx context.Context, pluginID string) ([]domain.TextVersion, error)
}

// ContentHashes is the hashes recorded for the files of one vault, used to select
// the live version of a file.
type ContentHashes interface {
	// FindContentHash returns the hash recorded for the file at the path, and false when none is.
	FindContentHash(ctx context.Context, path string) (text.ContentHash, bool, error)
	// SaveContentHash records the hash of a file on the index row of its path, where
	// the row has the record's stat or its hash. A file with no row is not recorded.
	SaveContentHash(ctx context.Context, record text.ContentHash) error
}

// VersionStores opens the versions of one vault to read them and to change them, and
// the content hashes recorded beside them.
type VersionStores interface {
	OpenVersionReader(v domain.Vault) (VersionReader, error)
	OpenVersionWriter(v domain.Vault) (VersionWriter, error)
	OpenContentHashes(v domain.Vault) (ContentHashes, error)
}

// TextFiles is the version files in the plugin folders of one vault's store.
type TextFiles interface {
	// ReadTextFile reads a file of a plugin's folder, or returns text.ErrNoText where it is not there.
	ReadTextFile(ctx context.Context, pluginID, name string) (text.File, error)
	// WriteTextFile writes the segments as a file of a plugin's folder and
	// returns the hash of its bytes. The file is replaced atomically.
	WriteTextFile(ctx context.Context, pluginID, name string, segments []domain.TextSegment) (string, error)
	// DeleteTextFile deletes a file of a plugin's folder. A file that is not there is deleted.
	DeleteTextFile(ctx context.Context, pluginID, name string) error
}

// TextFileStores opens the version files of one vault.
type TextFileStores interface {
	Open(v domain.Vault) (TextFiles, error)
}
