package assembly

import (
	machineport "github.com/jiva-studio/numen/libs/core/usecase/file/machinecontract"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
)

// SettingsAssembly is what this window asks to read and write settings.
type SettingsAssembly interface {
	// The use cases that read and write the settings.
	OpenGetSettings() *settings.Get
	OpenWriteSettings() *settings.Write
	// OpenChoosePluginPath makes the use case that puts the picker for a plugin's path setting in
	// front of the person.
	OpenChoosePluginPath(dialog machineport.PathDialog) *settings.ChoosePluginPath
}
