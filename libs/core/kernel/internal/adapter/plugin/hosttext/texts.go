package hosttext

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

// VersionWrites records and removes the versions plugins make of files.
type VersionWrites interface {
	PutVersion(ctx context.Context, v domain.Vault, pluginID string, params plugin.TextPut) (domain.TextVersion, error)
	RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error
}

// VersionReads reads the versions of files and their text.
type VersionReads interface {
	GetContentHash(ctx context.Context, v domain.Vault, path string) (string, error)
	ListVersions(ctx context.Context, v domain.Vault, path, contentHash string) (plugin.TextListing, error)
	GetTextPage(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (plugin.TextPage, error)
	GetTextRange(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, start, length int) (plugin.TextRange, error)
}

// Texts is the text versions of the vaults, for the plugins that call the host. Each call works
// the vault it is given. Which version is live is decided for the file as it is now.
type Texts struct {
	writes VersionWrites
	reads  VersionReads
}

// NewTexts returns the text versions of the vaults.
func NewTexts(writes VersionWrites, reads VersionReads) Texts {
	return Texts{writes: writes, reads: reads}
}

// PutVersion records a version a plugin made of a file of the vault.
func (p Texts) PutVersion(ctx context.Context, v domain.Vault, pluginID string, params plugin.TextPut) (domain.TextVersion, error) {
	return p.writes.PutVersion(ctx, v, pluginID, params)
}

// RemoveVersion removes a version a plugin made of a file of the vault.
func (p Texts) RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error {
	return p.writes.RemoveVersion(ctx, v, pluginID, path, key)
}

// ListVersions lists the versions of a file of the vault and names the live one for the file's
// current content.
func (p Texts) ListVersions(ctx context.Context, v domain.Vault, path string) (plugin.TextListing, error) {
	contentHash, err := p.reads.GetContentHash(ctx, v, path)
	if errors.Is(err, text.ErrFileUnreadable) {
		return plugin.TextListing{}, fmt.Errorf("%w: %w", text.ErrNoText, err)
	}
	if err != nil {
		return plugin.TextListing{}, err
	}
	return p.reads.ListVersions(ctx, v, path, contentHash)
}

// ReadText reads one page of the segments of a version of a file of the vault.
func (p Texts) ReadText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (plugin.TextPage, error) {
	return p.reads.GetTextPage(ctx, v, path, id, cursor)
}

// ReadTextRange reads the segments of a version of a file of the vault that cover a range of its
// whole text.
func (p Texts) ReadTextRange(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, start, length int) (plugin.TextRange, error) {
	return p.reads.GetTextRange(ctx, v, path, id, start, length)
}
