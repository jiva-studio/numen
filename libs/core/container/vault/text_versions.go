package vault

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/textfiles"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hosttext"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/textbridge"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// NewTextVersions is the text versions of the vaults' files: recorded in the
// index database, their files in the plugin folders of each vault's store,
// made by the enabled plugins that declare a text input.
func (c Config) NewTextVersions(db *Index) (*textversion.Versions, error) {
	var plugins textport.TextPlugins = textbridge.NoPlugins{}
	if c.PluginHost != nil && !c.SafeMode {
		plugins = textbridge.NewPlugins(c.PluginHost)
	}
	return textversion.NewVersions(db.TextVersions(), textfiles.Stores{Options: c.vaultOptions()}, plugins, c.VaultReaders(), c.Clock)
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

// SetTextVersionListener sets the listener notified of each change of the text versions.
func (c Config) SetTextVersionListener(listener textport.VersionListener) {
	if c.TextVersions != nil {
		c.TextVersions.SetListener(listener)
	}
}

// ReportTextFilesChanged tells the text versions that the files at paths changed.
func (c Config) ReportTextFilesChanged(ctx context.Context, v domain.Vault, paths []string) error {
	if c.TextVersions == nil {
		return nil
	}
	return c.TextVersions.ReportFilesChanged(ctx, v, paths)
}
