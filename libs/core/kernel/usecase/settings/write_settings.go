package settings

import (
	"errors"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// WriteSettings puts settings into the file. A value the settings cannot be read out of
// again is refused, and the file is left as it was.
type WriteSettings struct {
	check   *CheckSettings
	file    SettingsFile
	refresh SettingRefresher
}

// NewWriteSettings makes the use case. After a write, refresh tells the readers of the setting
// changes.
func NewWriteSettings(
	check *CheckSettings, file SettingsFile, refresh SettingRefresher,
) *WriteSettings {
	return &WriteSettings{check: check, file: file, refresh: refresh}
}

// WriteResult is every setting after a write, with the version of that state.
type WriteResult struct {
	Written string
	Version uint64
}

// Execute writes the settings. A plugin is given its new values after Execute returns.
func (u *WriteSettings) Execute(entries []settingsdomain.Setting) error {
	_, err := u.Write(entries)
	return err
}

// Write writes the settings like Execute and answers the settings as they stand after the write.
// The answer is the state this write left and its version, whatever other writes follow it.
// A write that was persisted and whose answer cannot be made returns an error that wraps
// settingsdomain.ErrWrittenNotAnswered, and the plugins are notified.
func (u *WriteSettings) Write(entries []settingsdomain.Setting) (WriteResult, error) {
	if err := u.check.Execute(entries); err != nil {
		return WriteResult{}, err
	}
	content, version, err := u.file.WriteJSON(entries)
	if err != nil && !errors.Is(err, settingsdomain.ErrWrittenNotAnswered) {
		return WriteResult{}, err
	}
	u.refresh.Refresh()
	if err != nil {
		return WriteResult{}, err
	}
	return WriteResult{Written: content, Version: version}, nil
}
