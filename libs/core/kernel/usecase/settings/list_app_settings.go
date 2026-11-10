package settings

import (
	"encoding/json"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

// ListAppSettings lists the settings the shell declares, each with the value in force and the
// options it offers.
type ListAppSettings struct {
	file       settingsport.SettingsFile
	defaults   settingsport.Defaults
	app        settingsport.AppDeclarations
	registries settingsport.RegistryOptions
}

// NewListAppSettings makes the use case. Where the file cannot be read, the values are the defaults.
func NewListAppSettings(
	file settingsport.SettingsFile, defaults settingsport.Defaults, app settingsport.AppDeclarations,
	registries settingsport.RegistryOptions,
) *ListAppSettings {
	return &ListAppSettings{file: file, defaults: defaults, app: app, registries: registries}
}

// Execute lists the settings in the order the shell declares them.
func (u *ListAppSettings) Execute() []plugin.Setting {
	current, err := u.file.ReadChecked()
	if err != nil {
		current = u.defaults.GetDefaults()
	}
	declared := u.app.GetAppDeclaration()
	entries := make([]plugin.Setting, 0, len(declared))
	for _, one := range declared {
		options := one.Options
		if one.OptionsFrom != "" {
			options, _ = u.registries.ListRegistryOptions(one.OptionsFrom)
		}
		defaultJSON := ""
		if encoded, err := json.Marshal(one.Default); err == nil && one.Default != nil {
			defaultJSON = string(encoded)
		}
		setting := plugin.NewSetting(one, defaultJSON, options)
		setting.ActiveOptions = options
		setting.IsShown = true
		path := append([]string{settingsdomain.AppOwner}, strings.Split(one.Key, ".")...)
		if value, found := current.GetValue(path...); found {
			if encoded, err := json.Marshal(value); err == nil {
				setting.ValueJSON = string(encoded)
			}
		}
		entries = append(entries, setting)
	}
	return entries
}
