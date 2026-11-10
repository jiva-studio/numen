package container

import (
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	settingswiring "github.com/jiva-studio/numen/libs/core/kernel/container/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

const agentAt = settingswiring.AgentAt

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() settingsadapter.File {
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
func (c Config) Settings() (settingsadapter.File, error) {
	return c.settings().Settings()
}

func (c Config) getSettingsAt(path string) (settingsadapter.File, error) {
	return c.settings().GetSettingsAt(path)
}

func (c Config) getSettingsOrDefaults() settingsadapter.File {
	return c.settings().GetSettingsOrDefaults()
}

// SetSettings returns the configuration with the values of a settings file
// applied. Both entry points use it.
func (c Config) SetSettings(s settingsadapter.File) Config {
	c.Hardware, c.ResolvedCompute = c.settings().ResolveCompute(s)
	c.settings().ConfigurePluginHost(s)
	return c
}

// Plugins returns a function that lists the discovered plugins and their
// settings schemas.
func (c Config) Plugins() func() []plugin.Info {
	return c.settings().Plugins()
}

// Models returns a function that lists the models each model setting accepts,
// and the programs the agent setting accepts.
func (c Config) Models() func() []settingsdomain.Model {
	return c.modelsForProvider(c.AgentProvider())
}

func (c Config) modelsForProvider(provider agentport.AgentProvider) func() []settingsdomain.Model {
	c.agentProvider = provider
	if c.getPrograms() == nil {
		c.programs = &settingsadapter.ProgramList{}
	}
	return c.settings().ModelsForProvider(c.agentProvider)
}

// SubscribeSettingChanges starts listening to the settings that change. See settingsport.ChangeSubscriber.
func (c Config) SubscribeSettingChanges() (<-chan []settingsdomain.Setting, func()) {
	return c.settings().SubscribeSettingChanges()
}

// SyncSetting returns whether note title and filename synchronization is enabled.
func (c Config) SyncSetting() query.SyncSetting {
	return c.settings().SyncSetting()
}

// ReadSettings returns a function that reads settings as JSON and returns the settings file path.
func (c Config) ReadSettings() func() (string, string, error) {
	return c.settings().ReadSettings()
}

// GetSettingsWriter patches specific settings paths in the configuration file. The readers of the
// setting-changed event receive the settings whose value changed.
func (c Config) GetSettingsWriter() func(entries []settingsdomain.Setting) error {
	return c.settings().GetSettingsWriter()
}

func (c Config) settingsFile() (string, error) {
	return c.settings().SettingsFile()
}

// Themes provides the themes the window can use: the catalog of themes, and
// the settings file that stores the selection.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	return c.settings().Themes(say)
}

// UnsubscribeHost stops the plugin host from receiving setting changes. It returns after the batch
// being delivered has been handled.
func (c Config) UnsubscribeHost() {
	c.settings().UnsubscribeHost()
}
