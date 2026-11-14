package settingsport

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// AppDeclarations are the settings the shell declares.
type AppDeclarations interface {
	GetAppDeclaration() []manifest.Setting
}

// PluginInfos lists the plugins and the settings each declares.
type PluginInfos interface {
	GetPluginInfos() []plugin.Info
}

// RegistryOptions lists the options a named registry offers to the settings that draw on it.
type RegistryOptions interface {
	// ListRegistryOptions returns the options, and false for a registry that is not known.
	ListRegistryOptions(name string) (manifest.SettingOptions, bool)
}
