package hosttext

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	textport "github.com/jiva-studio/numen/libs/core/usecase/textversion/textcontract"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// Texts is the text versions of the vaults, for the plugins that call the host. Each call works
// the vault it is given. Which version is live is decided for the file as it is now.
type Texts struct {
	versions *textversion.Versions
	readers  vaultport.VaultReaders
}

// NewTexts makes the text versions of the vaults, read through readers.
func NewTexts(versions *textversion.Versions, readers vaultport.VaultReaders) Texts {
	return Texts{versions: versions, readers: readers}
}

// PutVersion records a version a plugin made of a file of the vault.
func (p Texts) PutVersion(ctx context.Context, v domain.Vault, pluginID string, params pluginport.PluginTextPut) (domain.TextVersion, error) {
	return p.versions.PutVersion(ctx, v, pluginID, textversion.VersionPut{
		Path: params.Path, Key: params.Key, Source: params.Source, Kind: params.Kind,
		SourceHash: params.SourceHash, File: params.File,
	})
}

// RemoveVersion removes a version a plugin made of a file of the vault.
func (p Texts) RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error {
	return p.versions.RemoveVersion(ctx, v, pluginID, path, key)
}

// ListVersions lists the versions of a file of the vault and names the live one for the file's
// current content.
func (p Texts) ListVersions(ctx context.Context, v domain.Vault, path string) (pluginport.PluginTextListing, error) {
	reader, err := p.readers.Open(v)
	if err != nil {
		return pluginport.PluginTextListing{}, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return pluginport.PluginTextListing{}, fmt.Errorf("%w: %w", textport.ErrNoText, err)
	}
	listing, err := p.versions.ListVersions(ctx, v, path, domain.HashContent(raw))
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
