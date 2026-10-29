package container

import (
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/textfiles"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hosttext"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/textbridge"
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
	} = textbridge.NoPlugins{}
	if c.PluginHost != nil && !c.SafeMode {
		plugins = textbridge.NewPlugins(c.PluginHost)
	}
	return textversion.NewVersions(db.TextVersions(), textfiles.Stores{Options: c.vaultOptions()}, plugins, plugins, c.VaultReaders(), c.Clock())
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

// SetPluginTexts provides the text versions of the vaults to the plugins through
// the host. Without text versions, plugins are told there are none.
func (c Config) SetPluginTexts() {
	if c.PluginHost == nil {
		return
	}
	var host pluginport.PluginTextHost = c.PluginHost
	if c.TextVersions == nil {
		host.SetPluginTexts(nil)
		return
	}
	host.SetPluginTexts(hosttext.NewTexts(c.TextVersions, c.VaultReaders()))
}

// OpenPluginData is the folders plugins keep in a vault's store and the records of their text
// versions, and nil where the installation keeps no text versions.
func (c Config) openPluginData() *textversion.PluginData {
	if c.TextVersions == nil {
		return nil
	}
	data, err := textversion.NewPluginData(c.TextVersions, derivedfile.PluginStores{Options: c.vaultOptions()})
	if err != nil {
		c.handleError(err)
		return nil
	}
	return data
}
