package settingsport

import "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"

// SettingsFile is the file in which this installation is configured.
type SettingsFile interface {
	// ReadChecked returns the settings in force: a value that no declaration accepts is left out. A
	// file that is not there is read as the defaults and is not created.
	ReadChecked() (settings.Stored, error)
	// ReadUnchecked returns the settings as the file holds them. A file that is not there is
	// created with the defaults.
	ReadUnchecked() (settings.Stored, error)
	// Write puts each setting at its path in the file and leaves the rest of the file as it was.
	Write(entries []settings.Setting) error
}

// Defaults are the settings of an unconfigured installation.
type Defaults interface {
	GetDefaults() settings.Stored
}

// SettingRefresher tells the readers of the setting changes to look at the file now.
type SettingRefresher interface {
	Refresh()
}

// WriteValidator checks a value against the declaration of the setting it is written to.
type WriteValidator interface {
	// CheckWrite returns the reason the value is refused, and nil where it is accepted.
	CheckWrite(path []string, value any) error
}

// ValueCodec converts the value of a setting between the JSON the settings hold it as and the value
// it stands for.
type ValueCodec interface {
	// Decode returns the value the JSON stands for, or the reason it is no JSON.
	Decode(json string) (any, error)
	// Encode returns the JSON of the value.
	Encode(value any) (string, error)
}
