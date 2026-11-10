package settings

import settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"

// Write puts settings into the file. A value the settings cannot be read out of
// again is refused, and the file is left as it was.
type Write struct {
	write func(settings []settingsdomain.Setting) error
}

func NewWrite(write func(settings []settingsdomain.Setting) error) *Write {
	return &Write{write: write}
}

// Execute writes the settings. A plugin is given its new values after Execute returns.
func (u *Write) Execute(settings []settingsdomain.Setting) error {
	return u.write(settings)
}
