package assembly

import (
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
)

// SettingsAssembly is what this window asks to read and write settings and settings files.
type SettingsAssembly interface {
	// The use cases that read and write the settings and the file they stand in.
	OpenGetSettings() *settings.Get
	OpenWriteSettings() *settings.Write
	OpenGetSettingsFile() *settings.GetFile
	OpenWriteSettingsFile() *settings.WriteFile
	// OpenChoosePluginPath makes the use case that puts the picker for a plugin's path setting in
	// front of the person.
	OpenChoosePluginPath(dialog machineport.PathDialog) *settings.ChoosePluginPath
}
