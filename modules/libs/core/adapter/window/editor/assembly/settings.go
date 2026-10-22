package assembly

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

// SettingsAssembly is what this window asks to read and write settings and settings files.
type SettingsAssembly interface {
	// The use cases that read and write the settings and the file they stand in.
	OpenGetSettings() *settings.Get
	OpenWriteSettings() *settings.Write
	OpenGetSettingsFile() *settings.GetFile
	OpenWriteSettingsFile() *settings.WriteFile
}
