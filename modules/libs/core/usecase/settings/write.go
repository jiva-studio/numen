package settings

import "github.com/jiva-studio/numen/modules/libs/core/port"

// Write puts settings into the file. A value the settings cannot be read out of
// again is refused with port.ErrNotASetting and the file is left as it was.
type Write struct {
	write func(written []port.Setting) error
}

func NewWrite(write func(written []port.Setting) error) *Write {
	return &Write{write: write}
}

func (u *Write) Execute(written []port.Setting) error {
	return u.write(written)
}
