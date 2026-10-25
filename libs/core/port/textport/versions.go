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
	// ListVersions is the versions of every file of the vault.
	ListVersions(ctx context.Context) ([]domain.TextVersion, error)
	// SaveVersion records a version, replacing one of the same path and id in its place.
	SaveVersion(ctx context.Context, version domain.TextVersion) error
	// RemoveVersion removes a version. A version that is not there is removed.
	RemoveVersion(ctx context.Context, path string, id domain.TextVersionID) error
}

// VersionRepositories opens the versions of one vault.
type VersionRepositories interface {
	Open(v domain.Vault) (VersionRepository, error)
}

// VersionListener hears that the versions of a file changed: one was put,
// removed, or made live or dead by a plugin switched on or off.
type VersionListener interface {
	ReportVersionsChanged(v domain.Vault, path string)
}
