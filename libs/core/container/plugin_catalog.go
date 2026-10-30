package container

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/assembly"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/hintsource"
	"github.com/jiva-studio/numen/libs/plugin-catalog/localsource"
)

// GetPluginCatalog is the catalog of the plugins on disk, the disabled ones too, of the plugins
// that ship with the application, and of the sources the configuration names. It is nil for an
// installation that names no target or has no plugin host.
func (c Config) GetPluginCatalog() *catalog.Service {
	if c.PluginCatalog.Target == "" || c.PluginHost == nil {
		return nil
	}
	config := c.PluginCatalog
	config.Sources = append(
		[]catalog.Source{localsource.New(slices.Clone(c.PluginDirs), localsource.SettingsFunc(c.PluginHost.IsPluginEnabled))},
		append(slices.Clone(config.Sources), hintsource.NewBuiltIn())...)
	service, err := catalog.NewService(config)
	if err != nil {
		c.handleError(err)
		return nil
	}
	return service
}

// getEditorCatalog is the catalog as the editor window asks for it, and nil where there is none.
func (c Config) getEditorCatalog() assembly.PluginCatalog {
	service := c.GetPluginCatalog()
	if service == nil {
		return nil
	}
	return service
}
