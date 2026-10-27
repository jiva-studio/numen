package textport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// VersionRepository is the text versions of one vault's files. A version is
// named by its path and its id.
type VersionRepository interface {
	// GetVersions is the versions of one file, in the order they were first saved.
	GetVersions(ctx context.Context, path string) ([]domain.TextVersion, error)
	// SaveVersion records a version, replacing one of the same path and id in its place.
	SaveVersion(ctx context.Context, version domain.TextVersion) error
	// RemoveVersion removes a version. A version that is not there is removed.
	RemoveVersion(ctx context.Context, path string, id domain.TextVersionID) error
	// RemovePluginVersions removes every version a plugin made, and returns them.
	RemovePluginVersions(ctx context.Context, pluginID string) ([]domain.TextVersion, error)
}

// VersionRepositories opens the versions of one vault.
type VersionRepositories interface {
	Open(v domain.Vault) (VersionRepository, error)
}

// TextFile is the segments of a version file and the hash of its bytes.
type TextFile struct {
	Segments []domain.TextSegment
	Hash     string
}

// TextFiles is the version files in the plugin folders of one vault's store.
type TextFiles interface {
	// ReadTextFile reads a file of a plugin's folder, or returns ErrNoText where it is not there.
	ReadTextFile(ctx context.Context, pluginID, name string) (TextFile, error)
	// WriteTextFile writes the segments as a file of a plugin's folder and
	// returns the hash of its bytes. The file is replaced atomically.
	WriteTextFile(ctx context.Context, pluginID, name string, segments []domain.TextSegment) (string, error)
	// DeleteTextFile removes a file of a plugin's folder. A file that is not there is removed.
	DeleteTextFile(ctx context.Context, pluginID, name string) error
}

// TextFileStores opens the version files of one vault.
type TextFileStores interface {
	Open(v domain.Vault) (TextFiles, error)
}

// VersionListener is told that the versions of a file changed: one was put,
// removed or moved, so the live version may differ.
type VersionListener interface {
	ReportVersionsChanged(v domain.Vault, path string)
}
