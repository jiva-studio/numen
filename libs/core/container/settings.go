package container

import (
	settingsadapter "github.com/jiva-studio/numen/libs/core/adapter/settings"
	settingswiring "github.com/jiva-studio/numen/libs/core/container/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/theme"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
)

const agentAt = settingswiring.AgentAt

// Settings is the file that configures this installation: one object whose
// fields are its sections, in file order.
type Settings = settingswiring.Settings

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() Settings {
	return settingswiring.DefaultSettings()
}

// CheckInterfaceScale returns an error when scale is outside the interface scale bounds.
func CheckInterfaceScale(name string, scale float64) error {
	return settingswiring.CheckInterfaceScale(name, scale)
}

// CheckTextScale returns an error when scale is outside the text scale bounds.
func CheckTextScale(name string, scale float64) error {
	return settingswiring.CheckTextScale(name, scale)
}

// Settings reads the settings file. An unconfigured installation returns the
// default settings.
func (c Config) Settings() (Settings, error) {
	return c.settings().Settings()
}

func (c Config) getSettingsAt(path string) (Settings, error) {
	return c.settings().GetSettingsAt(path)
}

func (c Config) getSettingsOrDefaults() Settings {
	return c.settings().GetSettingsOrDefaults()
}

// SetSettings returns the configuration with the values of a settings file
// applied. Both entry points use it.
func (c Config) SetSettings(s Settings) Config {
	c.Hardware, c.ResolvedCompute = c.settings().ResolveCompute(s)
	c.settings().ConfigurePluginHost(s)
	return c
}

// Plugins returns a function that lists the discovered plugins and their
// settings schemas.
func (c Config) Plugins() func() []pluginport.PluginInfo {
	return c.settings().Plugins()
}

// Models returns a function that lists the models each model setting accepts,
// and the programs the agent setting accepts.
func (c Config) Models() func() []settingsport.Model {
	return c.modelsForProvider(c.AgentProvider())
}

func (c Config) modelsForProvider(provider agentport.AgentProvider) func() []settingsport.Model {
	c.agentProvider = provider
	c.programs = &settingsadapter.ProgramList{}
	return c.settings().ModelsForProvider(c.agentProvider)
}

func (c Config) listAppSettingsOn(goos string) []pluginport.PluginSetting {
	return c.settings().ListAppSettingsOn(goos)
}

// SubscribeSettingChanges starts listening to the settings that change. See settingsport.ChangeSubscriber.
func (c Config) SubscribeSettingChanges() (<-chan []settingsport.Setting, func()) {
	return c.settings().SubscribeSettingChanges()
}

// SyncSetting returns whether note title and filename synchronization is enabled.
func (c Config) SyncSetting() note.SyncSetting {
	return c.settings().SyncSetting()
}

// ReadSettings returns a function that reads settings as JSON and returns the settings file path.
func (c Config) ReadSettings() func() (string, string, error) {
	return c.settings().ReadSettings()
}

// GetSettingsWriter patches specific settings paths in the configuration file. The readers of the
// setting-changed event receive the settings whose value changed.
func (c Config) GetSettingsWriter() func(entries []settingsport.Setting) error {
	return c.settings().GetSettingsWriter()
}

func (c Config) settingsFile() (string, error) {
	return c.settings().SettingsFile()
}

// Themes provides the themes the window can use: the catalogue of themes, and
// the settings file that stores the selection.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	return c.settings().Themes(say)
}
