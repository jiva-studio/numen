package textport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Texts is the text of one vault's files, as the index, the search and the
// highlighting read it: the text of a version, read from the plugin that made
// it. fileHash is domain.HashContent of the file as it is now.
type Texts interface {
	// FindLiveVersion is the file's live version, or ErrNoText where none is live.
	FindLiveVersion(ctx context.Context, path, fileHash string) (domain.TextVersionID, error)
	// GetLiveText is the text of the file's live version, or ErrNoText where none is live.
	GetLiveText(ctx context.Context, path, fileHash string) (domain.VersionText, error)
	// GetVersionText is the text of one version, live or not, or ErrNoText.
	GetVersionText(ctx context.Context, path string, id domain.TextVersionID) (domain.VersionText, error)
}

// TextReaders opens the text of one vault.
type TextReaders interface {
	Open(v domain.Vault) (VaultTexts, error)
}

// VaultTexts is the text of one vault, read, and asked for where a file has none.
type VaultTexts interface {
	Texts
	// RequestText asks every enabled plugin that runs on its own and takes the
	// file for text of it. A file with a live version is asked for nothing.
	RequestText(ctx context.Context, path, fileHash string) error
	// IsTextSource reports whether the index keeps the text of the file: it has
	// text versions, or an enabled plugin that runs on its own takes it.
	IsTextSource(ctx context.Context, path string) (bool, error)
}
