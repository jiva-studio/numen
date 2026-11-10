package hosttext

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
)

// Texts is the text versions of the vaults, for the plugins that call the host. Each call works
// the vault it is given. Which version is live is decided for the file as it is now.
type Texts struct {
	versions *textversion.Versions
}

// NewTexts returns the text versions of the vaults.
func NewTexts(versions *textversion.Versions) Texts {
	return Texts{versions: versions}
}

// PutVersion records a version a plugin made of a file of the vault.
func (p Texts) PutVersion(ctx context.Context, v domain.Vault, pluginID string, params plugin.TextPut) (domain.TextVersion, error) {
	return p.versions.PutVersion(ctx, v, pluginID, textversion.VersionPut{
		Path: params.Path, Key: params.Key, Source: params.Source, SourceProperty: params.SourceProperty,
		Kind: params.Kind, InputHash: params.InputHash, File: params.File,
	})
}

// RemoveVersion removes a version a plugin made of a file of the vault.
func (p Texts) RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error {
	return p.versions.RemoveVersion(ctx, v, pluginID, path, key)
}

// ListVersions lists the versions of a file of the vault and names the live one for the file's
// current content.
func (p Texts) ListVersions(ctx context.Context, v domain.Vault, path string) (plugin.TextListing, error) {
	contentHash, err := p.versions.GetContentHash(ctx, v, path)
	if errors.Is(err, textversion.ErrFileUnreadable) {
		return plugin.TextListing{}, fmt.Errorf("%w: %w", text.ErrNoText, err)
	}
	if err != nil {
		return plugin.TextListing{}, err
	}
	listing, err := p.versions.ListVersions(ctx, v, path, contentHash)
	if err != nil {
		return plugin.TextListing{}, err
	}
	return plugin.TextListing{Versions: listing.Versions, Live: listing.Live}, nil
}

// ReadText reads one page of the segments of a version of a file of the vault.
func (p Texts) ReadText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (plugin.TextPage, error) {
	page, err := p.versions.GetTextPage(ctx, v, path, id, cursor)
	if err != nil {
		return plugin.TextPage{}, err
	}
	return plugin.TextPage{Segments: page.Segments, NextCursor: page.NextCursor}, nil
}

// ReadTextRange reads the segments of a version of a file of the vault that cover a range of its
// whole text.
func (p Texts) ReadTextRange(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, start, length int) (plugin.TextRange, error) {
	textRange, err := p.versions.GetTextRange(ctx, v, path, id, start, length)
	if err != nil {
		return plugin.TextRange{}, err
	}
	return plugin.TextRange{Segments: textRange.Segments, Start: textRange.Start, Size: textRange.Size}, nil
}
