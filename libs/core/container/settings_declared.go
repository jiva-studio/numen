package container

import (
	"encoding/json"
	"strings"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// getDeclarations returns the settings each owner declares: the shell, and the plugins of the host.
func (c Config) getDeclarations() settings.Declarations {
	return func(owner string) ([]pluginport.PluginManifestSetting, bool) {
		if owner == settings.OwnerApp {
			return settings.GetAppDeclaration(), true
		}
		if c.PluginHost == nil {
			return nil, false
		}
		return c.PluginHost.GetPluginSettingDeclarations(owner)
	}
}

// getRegistries returns the registries an option list may name: the installed themes.
func (c Config) getRegistries() settings.Registries {
	return func(name string) ([]string, bool) {
		options, found := c.listRegistryOptions(name)
		names := make([]string, 0, len(options))
		for _, one := range options {
			names = append(names, one.Value)
		}
		return names, found
	}
}

// listRegistryOptions returns the options a registry lists, and false for a registry that has no
// provider.
func (c Config) listRegistryOptions(name string) (pluginport.PluginSettingOptions, bool) {
	if name != settings.RegistryThemes {
		return nil, false
	}
	catalogue, _ := c.getCatalogue()
	var options pluginport.PluginSettingOptions
	for _, one := range catalogue.Themes() {
		options = append(options, pluginport.PluginSettingOption{Value: one.Name, Title: one.Title})
	}
	return options, true
}

// listAppSettings returns the declaration of the settings of the shell with the options, the
// visibility and the value in force of each under the settings file.
func (c Config) listAppSettings() []pluginport.PluginSetting {
	current := c.getSettingsOrDefaults()
	declared := settings.GetAppDeclaration()
	entries := make([]pluginport.PluginSetting, 0, len(declared))
	for _, one := range declared {
		options := one.Options
		if one.OptionsFrom != "" {
			options, _ = c.listRegistryOptions(one.OptionsFrom)
		}
		defaultJSON := ""
		if encoded, err := json.Marshal(one.Default); err == nil && one.Default != nil {
			defaultJSON = string(encoded)
		}
		setting := pluginport.NewPluginSetting(one, defaultJSON, options)
		setting.ActiveOptions = options
		setting.IsShown = true
		if value, found := current.GetValue(append([]string{settings.OwnerApp}, strings.Split(one.Key, ".")...)...); found {
			if encoded, err := json.Marshal(value); err == nil {
				setting.ValueJSON = string(encoded)
			}
		}
		entries = append(entries, setting)
	}
	return entries
}
