package settings

import "github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"

type (
	PluginSetting          = pluginport.PluginSetting
	PluginInfo             = pluginport.PluginInfo
	PluginFailure          = pluginport.PluginFailure
	PluginReaderInfo       = pluginport.PluginReaderInfo
	SecretDeclaration      = pluginport.SecretDeclaration
	Secret                 = pluginport.Secret
	DeclarationKey         = pluginport.DeclarationKey
	PluginSettingsView     = pluginport.PluginSettingsView
	PluginSettingOption    = pluginport.PluginSettingOption
	PluginSettingOptions   = pluginport.PluginSettingOptions
	PluginSettingOptionsBy = pluginport.PluginSettingOptionsBy
	PluginSettingCondition = pluginport.PluginSettingCondition
	UnavailableError       = pluginport.UnavailableError
)

// A plugin the host cannot reach is not installed, disabled or failed.
const PluginNotInstalled = pluginport.PluginNotInstalled
