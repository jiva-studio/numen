package assembly

import (
	machineport "github.com/jiva-studio/numen/libs/core/usecase/file/machinecontract"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
)

// SettingsAssembly provides what this window needs to read and write settings.
type SettingsAssembly interface {
	// The use cases that read and write the settings.
	OpenGetSettings() *settings.Get
	OpenWriteSettings() *settings.Write
	// OpenChoosePluginPath creates the use case that shows the picker for a plugin's path setting.
	OpenChoosePluginPath(dialog machineport.PathDialog) *settings.ChoosePluginPath
}
