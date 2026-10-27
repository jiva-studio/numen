package container

import (
	"context"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/adapter/index/version"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/textfiles"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hosttext"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/textproviderbridge"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// coreFolder is the folder of a vault's store that holds the core's own files.
const coreFolder = "core"

// NewTextVersions is the text versions of the vaults' files: recorded in the
// index database, their files in the plugin folders of each vault's store,
// made by the enabled plugins that declare a text input.
func (c Config) NewTextVersions(db *Index) (*textversion.Versions, error) {
	var plugins interface {
		textport.TextPlugins
		textport.TextProviders
	} = noTextPlugins{}
	if c.PluginHost != nil && !c.SafeMode {
		plugins = textproviderbridge.NewPlugins(c.PluginHost)
	}
	repos := indexVersions{repo: db.db.TextVersions()}
	return textversion.NewVersions(repos, textfiles.Stores{Options: c.vaultOptions()}, plugins, plugins, c.Clock())
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

// SetPluginTexts is the text versions of the vault getVault names, as the
// plugins work them through the host. Without text versions a plugin is told
// there are none.
func (c Config) SetPluginTexts(_ context.Context, getVault func() (domain.Vault, bool)) {
	if c.PluginHost == nil {
		return
	}
	var host pluginport.PluginTextHost = c.PluginHost
	if c.TextVersions == nil {
		host.SetPluginTexts(nil)
		return
	}
	host.SetPluginTexts(hosttext.NewTexts(c.TextVersions, c.VaultReaders(), getVault))
}

// indexVersions is the text versions the index database records, one vault at a time.
type indexVersions struct {
	repo *version.Repository
}

func (r indexVersions) Open(v domain.Vault) (textport.VersionRepository, error) {
	return vaultVersions{repo: r.repo, vault: v.ID}, nil
}

type vaultVersions struct {
	repo  *version.Repository
	vault domain.VaultID
}

func (r vaultVersions) GetVersions(ctx context.Context, path string) ([]domain.TextVersion, error) {
	return r.repo.GetVersions(ctx, r.vault, path)
}

func (r vaultVersions) SaveVersion(ctx context.Context, v domain.TextVersion) error {
	return r.repo.SaveVersion(ctx, r.vault, v)
}

func (r vaultVersions) RemoveVersion(ctx context.Context, path string, id domain.TextVersionID) error {
	pluginID, key := splitVersionID(id)
	return r.repo.RemoveVersion(ctx, r.vault, path, pluginID, key)
}

func (r vaultVersions) RemovePluginVersions(ctx context.Context, pluginID string) ([]domain.TextVersion, error) {
	return r.repo.RemovePluginVersions(ctx, r.vault, pluginID)
}

// splitVersionID is the plugin and the key of a version id. A plugin id holds
// no slash, so the first slash ends it.
func splitVersionID(id domain.TextVersionID) (pluginID, key string) {
	pluginID, key, _ = strings.Cut(string(id), "/")
	return pluginID, key
}

// noTextPlugins is an installation with no plugin that makes text.
type noTextPlugins struct{}

func (noTextPlugins) GetTextPlugins() []textport.TextPlugin { return nil }

func (noTextPlugins) OpenProvider(_ context.Context, pluginID string) (textport.TextProvider, error) {
	return nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginNotInstalled}
}

// OpenPluginData is the folders plugins keep in a vault's store and the records of their text
// versions, and nil where the installation keeps no text versions.
func (e EditorAssembly) OpenPluginData() *textversion.PluginData {
	if e.cfg.TextVersions == nil {
		return nil
	}
	data, err := textversion.NewPluginData(e.cfg.TextVersions, derivedfile.PluginStores{Options: e.cfg.vaultOptions()})
	if err != nil {
		e.cfg.handleError(err)
		return nil
	}
	return data
}
