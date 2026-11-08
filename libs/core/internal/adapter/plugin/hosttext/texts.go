package hosttext

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	textport "github.com/jiva-studio/numen/libs/core/usecase/textversion/textcontract"
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
func (p Texts) PutVersion(ctx context.Context, v domain.Vault, pluginID string, params pluginport.PluginTextPut) (domain.TextVersion, error) {
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
func (p Texts) ListVersions(ctx context.Context, v domain.Vault, path string) (pluginport.PluginTextListing, error) {
	contentHash, err := p.versions.GetContentHash(ctx, v, path)
	if errors.Is(err, textversion.ErrFileUnreadable) {
		return pluginport.PluginTextListing{}, fmt.Errorf("%w: %w", textport.ErrNoText, err)
	}
	if err != nil {
		return pluginport.PluginTextListing{}, err
	}
	listing, err := p.versions.ListVersions(ctx, v, path, contentHash)
	if err != nil {
		return pluginport.PluginTextListing{}, err
	}
	return pluginport.PluginTextListing{Versions: listing.Versions, Live: listing.Live}, nil
}

// ReadText reads one page of the segments of a version of a file of the vault.
func (p Texts) ReadText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (pluginport.PluginTextPage, error) {
	page, err := p.versions.GetTextPage(ctx, v, path, id, cursor)
	if err != nil {
		return pluginport.PluginTextPage{}, err
	}
	return pluginport.PluginTextPage{Segments: page.Segments, NextCursor: page.NextCursor}, nil
}

// ReadTextRange reads the segments of a version of a file of the vault that cover a range of its
// whole text.
func (p Texts) ReadTextRange(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, start, length int) (pluginport.PluginTextRange, error) {
	textRange, err := p.versions.GetTextRange(ctx, v, path, id, start, length)
	if err != nil {
		return pluginport.PluginTextRange{}, err
	}
	return pluginport.PluginTextRange{Segments: textRange.Segments, Start: textRange.Start, Size: textRange.Size}, nil
}
