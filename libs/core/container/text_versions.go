package container

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/versionfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/textproviderbridge"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// coreFolder is the folder of a vault's store that holds the core's own files.
const coreFolder = "core"

// NewTextVersions is the text versions of the vaults' files: kept in the core's
// folder of each vault's store, made by the enabled plugins that declare a text
// input, and read from their providers.
func (c Config) NewTextVersions() (*textversion.Versions, error) {
	repos := versionfile.NewRepositories(func(v domain.Vault) (versionfile.Folder, error) {
		store, err := derivedfile.OpenDerived(v.Path, c.vaultOptions(), coreFolder)
		if err != nil {
			return nil, err
		}
		return storeFolder{store: store}, nil
	})
	var plugins interface {
		textport.TextPlugins
		textport.TextProviders
	} = noTextPlugins{}
	if c.PluginHost != nil && !c.SafeMode {
		plugins = textproviderbridge.NewPlugins(c.PluginHost)
	}
	return textversion.NewVersions(repos, plugins, plugins, c.Clock())
}

// getTextReaders is the text versions sources are read from, or nil where the
// configuration has none.
func (c Config) getTextReaders() textport.TextReaders {
	if c.TextReaders != nil {
		return c.TextReaders
	}
	if c.TextVersions == nil {
		return nil
	}
	return c.TextVersions
}

// SetPluginTexts is the text versions of the vault getVault names, as the plugins
// work them through the host, and a plugin enabled or disabled changes which of
// that vault's versions are live. Without text versions a plugin is told there
// are none.
func (c Config) SetPluginTexts(ctx context.Context, getVault func() (domain.Vault, bool)) {
	if c.PluginHost == nil {
		return
	}
	var host pluginport.PluginTextHost = c.PluginHost
	if c.TextVersions == nil {
		host.SetPluginTexts(nil)
		return
	}
	versions := c.TextVersions
	host.SetPluginTexts(pluginTexts{versions: versions, readers: c.VaultReaders(), getVault: getVault})
	host.SetOnEnabledChanged(func(pluginID string) {
		v, ok := getVault()
		if !ok {
			return
		}
		if err := versions.ReportPluginChanged(ctx, v, pluginID); err != nil {
			c.handleError(err)
		}
	})
}

// storeFolder is the core's folder of one vault's store.
type storeFolder struct {
	store *derivedfile.DerivedStore
}

func (f storeFolder) Read(ctx context.Context, name string) ([]byte, error) {
	return f.store.Read(ctx, coreFolder+"/"+name)
}

func (f storeFolder) Write(ctx context.Context, name string, content []byte) error {
	return f.store.Write(ctx, coreFolder+"/"+name, content)
}

// noTextPlugins is an installation with no plugin that makes text.
type noTextPlugins struct{}

func (noTextPlugins) GetTextPlugins() []textport.TextPlugin { return nil }

func (noTextPlugins) OpenProvider(_ context.Context, pluginID string) (textport.TextProvider, error) {
	return nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginNotInstalled}
}

// pluginTexts is the text versions of the vault the host shows, for the plugins
// that call the host. Which version is live is asked of the file as it is now.
type pluginTexts struct {
	versions *textversion.Versions
	readers  vaultport.VaultReaders
	getVault func() (domain.Vault, bool)
}

// errNoVaultShown is a call about text versions while the host shows no vault.
var errNoVaultShown = fmt.Errorf("%w: the host shows no vault", textport.ErrNoText)

func (p pluginTexts) getShownVault() (domain.Vault, error) {
	v, ok := p.getVault()
	if !ok {
		return domain.Vault{}, errNoVaultShown
	}
	return v, nil
}

func (p pluginTexts) PutVersion(ctx context.Context, pluginID string, params pluginport.PluginTextPut) (domain.TextVersion, error) {
	v, err := p.getShownVault()
	if err != nil {
		return domain.TextVersion{}, err
	}
	return p.versions.PutVersion(ctx, v, pluginID, textversion.VersionPut{
		Path: params.Path, Key: params.Key, Input: params.Input, Format: params.Format, InputHash: params.InputHash, Hash: params.Hash,
	})
}

func (p pluginTexts) RemoveVersion(ctx context.Context, pluginID, path, key string) error {
	v, err := p.getShownVault()
	if err != nil {
		return err
	}
	return p.versions.RemoveVersion(ctx, v, pluginID, path, key)
}

func (p pluginTexts) ListVersions(ctx context.Context, path string) (pluginport.PluginTextListing, error) {
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

func (p pluginTexts) ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (pluginport.PluginTextPage, error) {
	v, err := p.getShownVault()
	if err != nil {
		return pluginport.PluginTextPage{}, err
	}
	page, err := p.versions.GetTextPage(ctx, v, path, id, cursor)
	if err != nil {
		return pluginport.PluginTextPage{}, err
	}
	return pluginport.PluginTextPage{Parts: page.Parts, NextCursor: page.NextCursor}, nil
}
