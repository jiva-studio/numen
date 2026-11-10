package settings

import (
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ThemeRegistry answers the registry of themes with the themes list returns, and every other registry
// as unknown.
func ThemeRegistry(list func() []themedomain.Theme) RegistryOptions {
	return func(name string) (manifest.SettingOptions, bool) {
		if name != RegistryThemes {
			return nil, false
		}
		var options manifest.SettingOptions
		for _, one := range list() {
			options = append(options, manifest.SettingOption{Value: one.Name, Title: one.Title})
		}
		return options, true
	}
}

// GetNames returns the registries by the values their options hold.
func (r RegistryOptions) GetNames() Registries {
	return func(name string) ([]string, bool) {
		options, found := r(name)
		names := make([]string, 0, len(options))
		for _, one := range options {
			names = append(names, one.Value)
		}
		return names, found
	}
}
