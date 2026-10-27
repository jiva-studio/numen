package hosttext

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// Texts is the text versions of the vault the host shows, for the plugins
// that call the host. Which version is live is decided for the file as it is now.
type Texts struct {
	versions *textversion.Versions
	readers  vaultport.VaultReaders
	getVault func() (domain.Vault, bool)
}

// NewTexts makes the text versions of the vault getVault names, read through readers.
func NewTexts(versions *textversion.Versions, readers vaultport.VaultReaders, getVault func() (domain.Vault, bool)) Texts {
	return Texts{versions: versions, readers: readers, getVault: getVault}
}

// errNoVaultShown is a call about text versions while the host shows no vault.
var errNoVaultShown = fmt.Errorf("%w: the host shows no vault", textport.ErrNoText)

func (p Texts) getShownVault() (domain.Vault, error) {
	v, ok := p.getVault()
	if !ok {
		return domain.Vault{}, errNoVaultShown
	}
	return v, nil
}

// PutVersion records a version a plugin made of a file of the shown vault.
func (p Texts) PutVersion(ctx context.Context, pluginID string, params pluginport.PluginTextPut) (domain.TextVersion, error) {
	v, err := p.getShownVault()
	if err != nil {
		return domain.TextVersion{}, err
	}
	return p.versions.PutVersion(ctx, v, pluginID, textversion.VersionPut{
		Path: params.Path, Key: params.Key, Source: params.Source, Kind: params.Kind,
		SourceHash: params.SourceHash, File: params.File,
	})
}

// RemoveVersion removes a version a plugin made of a file of the shown vault.
func (p Texts) RemoveVersion(ctx context.Context, pluginID, path, key string) error {
	v, err := p.getShownVault()
	if err != nil {
		return err
	}
	return p.versions.RemoveVersion(ctx, v, pluginID, path, key)
}

// ListVersions lists the versions of a file of the shown vault and names the
// live one for the file's current content.
func (p Texts) ListVersions(ctx context.Context, path string) (pluginport.PluginTextListing, error) {
	v, err := p.getShownVault()
	if err != nil {
		return pluginport.PluginTextListing{}, err
	}
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

// ReadText reads one page of the segments of a version of a file of the shown vault.
func (p Texts) ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (pluginport.PluginTextPage, error) {
	v, err := p.getShownVault()
	if err != nil {
		return pluginport.PluginTextPage{}, err
	}
	page, err := p.versions.GetTextPage(ctx, v, path, id, cursor)
	if err != nil {
		return pluginport.PluginTextPage{}, err
	}
	return pluginport.PluginTextPage{Segments: page.Segments, NextCursor: page.NextCursor}, nil
}
