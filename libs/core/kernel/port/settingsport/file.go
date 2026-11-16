package settingsport

import (
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// ErrWrittenNotAnswered is the error of a write that is persisted and whose answer cannot be made.
var ErrWrittenNotAnswered = errors.New("the settings are written and cannot be answered")

// SettingsFile is the file in which this installation is configured.
type SettingsFile interface {
	// ReadChecked returns the settings in force: a value that no declaration accepts is left out. A
	// file that is not there is read as the defaults and is not created.
	ReadChecked() (settings.Stored, error)
	// ReadUnchecked returns the settings as the file holds them. A file that is not there is
	// created with the defaults.
	ReadUnchecked() (settings.Stored, error)
	// WriteJSON puts each setting at its path in the file, leaves the rest of the file as it was, and
	// returns the settings in force as JSON with the version, both of this write and of no other. A
	// write that is persisted and whose answer cannot be made returns an error that wraps
	// ErrWrittenNotAnswered.
	WriteJSON(entries []settings.Setting) (content string, version uint64, err error)
}

// Defaults are the settings of an unconfigured installation.
type Defaults interface {
	GetDefaults() settings.Stored
}

// SettingRefresher tells the readers of the setting changes to look at the file now.
type SettingRefresher interface {
	Refresh()
}

// ValueCodec converts the value of a setting between the JSON the settings hold it as and the value
// it stands for.
type ValueCodec interface {
	// Decode returns the value the JSON stands for, or the reason it is no JSON.
	Decode(json string) (any, error)
	// Encode returns the JSON of the value.
	Encode(value any) (string, error)
}
