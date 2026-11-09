package assembly

import (
	machineport "github.com/jiva-studio/numen/libs/core/kernel/usecase/file/machinecontract"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

// SettingsAssembly provides what this window needs to read and write settings.
type SettingsAssembly interface {
	// The use cases that read and write the settings.
	OpenGetSettings() *settings.Get
	OpenWriteSettings() *settings.Write
	// OpenWatchSettings creates the use case that reports each setting that changes.
	OpenWatchSettings() *settings.Watch
	// OpenChoosePluginPath creates the use case that shows the picker for a plugin's path setting.
	OpenChoosePluginPath(dialog machineport.PathDialog) *settings.ChoosePluginPath
}
