package text

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Texts is the text of one vault's files, as the index, the search and the
// highlighting read it: the text of a version, read from the plugin that made
// it. contentHash is domain.HashContent of the file as it is now.
type Texts interface {
	// FindLiveVersion is the file's live version, or ErrNoText where none is live.
	FindLiveVersion(ctx context.Context, path, contentHash string) (domain.TextVersionID, error)
	// GetLiveText is the text of the file's live version, or ErrNoText where none is live.
	GetLiveText(ctx context.Context, path, contentHash string) (domain.VersionText, error)
	// GetVersionText is the text of one version, live or not, or ErrNoText.
	GetVersionText(ctx context.Context, path string, id domain.TextVersionID) (domain.VersionText, error)
}

// Readers opens the text of one vault.
type Readers interface {
	Open(v domain.Vault) (VaultTexts, error)
}

type VaultTexts interface {
	Texts
	// GetContentHash returns domain.HashContent of the file as it is now.
	GetContentHash(ctx context.Context, path string) (string, error)
	// IsTextSource reports whether the index keeps the text of the file: it has
	// text versions.
	IsTextSource(ctx context.Context, path string) (bool, error)
}

// VersionListener is notified that the versions of a file changed: one was added,
// removed or moved, so the live version may differ.
type VersionListener interface {
	ReportVersionsChanged(v domain.Vault, path string)
}
