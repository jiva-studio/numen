package settings

import (
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

// WriteSettings puts settings into the file. A value the settings cannot be read out of
// again is refused, and the file is left as it was.
type WriteSettings struct {
	check   *CheckSettings
	file    settingsport.SettingsFile
	refresh settingsport.SettingRefresher
}

// NewWriteSettings makes the use case. After a write, refresh tells the readers of the setting
// changes.
func NewWriteSettings(
	check *CheckSettings, file settingsport.SettingsFile, refresh settingsport.SettingRefresher,
) *WriteSettings {
	return &WriteSettings{check: check, file: file, refresh: refresh}
}

// Execute writes the settings. A plugin is given its new values after Execute returns.
func (u *WriteSettings) Execute(entries []settingsdomain.Setting) error {
	if err := u.check.Execute(entries); err != nil {
		return err
	}
	if err := u.file.Write(entries); err != nil {
		return err
	}
	u.refresh.Refresh()
	return nil
}
