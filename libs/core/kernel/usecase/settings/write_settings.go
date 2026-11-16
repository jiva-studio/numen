package settings

import (
	"fmt"

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

// Applied is every setting after a write, with the version of that state.
type Applied struct {
	Written string
	Version uint64
}

// Execute writes the settings. A plugin is given its new values after Execute returns.
func (u *WriteSettings) Execute(entries []settingsdomain.Setting) error {
	_, err := u.Write(entries)
	return err
}

// Write writes the settings like Execute and answers the settings as they stand after the write.
// The version is greater than that of any state read before the write.
// A write that was persisted and cannot be read back answers an error, and the plugins are told of the change.
func (u *WriteSettings) Write(entries []settingsdomain.Setting) (Applied, error) {
	if err := u.check.Execute(entries); err != nil {
		return Applied{}, err
	}
	if err := u.file.Write(entries); err != nil {
		return Applied{}, err
	}
	u.refresh.Refresh()
	content, _, version, err := u.file.ReadJSON()
	if err != nil {
		return Applied{}, fmt.Errorf("the settings are written and cannot be read back: %w", err)
	}
	return Applied{Written: content, Version: version}, nil
}
