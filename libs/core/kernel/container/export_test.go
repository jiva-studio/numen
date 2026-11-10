package container

import (
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/theme"
)

// Settings reads the settings file; an unconfigured installation returns the default settings.
func (c Config) Settings() (settingsadapter.File, error) { return c.readSettingsFile() }

// SetSettings returns the configuration with the values of a settings file applied.
func (c Config) SetSettings(s settingsadapter.File) Config { return c.applySettingsFile(s) }

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() settingsadapter.File { return defaultSettings() }

// Themes returns the themes service the window offers.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	return settingsThemes(c.settings(), say)
}
