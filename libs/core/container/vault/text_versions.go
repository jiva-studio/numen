package vault

import (
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/textfiles"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hosttext"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/textbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

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
	return textversion.NewVersions(db.TextVersions(), textfiles.Stores{Options: c.vaultOptions()}, plugins, plugins, c.VaultReaders(), c.Clock)
}

// GetTextReaders returns the text readers configured for this installation.
func (c Config) GetTextReaders() textport.TextReaders {
	if c.TextReaders != nil {
		return c.TextReaders
	}
	if c.TextVersions == nil {
		return nil
	}
	return c.TextVersions
}

// SetPluginTexts provides the text versions of the vaults to the plugins through
// the host. Without text versions, plugins are informed there are none.
func (c Config) SetPluginTexts() {
	if c.PluginHost == nil {
		return
	}
	var host pluginport.PluginTextHost = c.PluginHost
	if c.TextVersions == nil {
		host.SetPluginTexts(nil)
		return
	}
	host.SetPluginTexts(hosttext.NewTexts(c.TextVersions))
}

// OpenPluginData builds the plugin data cleaner for the active text versions.
func (c Config) OpenPluginData() *textversion.PluginData {
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

// OpenFileMenu builds the file menu over text versions and plugin commands.
func (c Config) OpenFileMenu() *textversion.FileMenu {
	if c.TextVersions == nil {
		return nil
	}
	var commands pluginport.PluginCommands = commandbridge.NoCommands{}
	var types contenttypeport.ContentTypeTable = sharedmimeinfo.Open()
	if c.PluginHost != nil && !c.SafeMode {
		commands, types = commandbridge.Open(c.PluginHost), c.PluginHost
	}
	menu, err := textversion.NewFileMenu(c.TextVersions, commands, types)
	if err != nil {
		c.handleError(err)
		return nil
	}
	return menu
}

// OpenFileText builds the file text reader over text versions.
func (c Config) OpenFileText() *textversion.FileText {
	if c.TextVersions == nil {
		return nil
	}
	text, err := textversion.NewFileText(c.TextVersions)
	if err != nil {
		c.handleError(err)
		return nil
	}
	return text
}
