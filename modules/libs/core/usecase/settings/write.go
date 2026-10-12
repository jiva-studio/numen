package settings

// Write puts settings into the file. A value the settings cannot be read out of
// again is refused with ErrNotASetting and the file is left as it was.
type Write struct {
	write func(written []Setting) error
}

func NewWrite(write func(written []Setting) error) *Write {
	return &Write{write: write}
}

func (u *Write) Execute(written []Setting) error {
	return u.write(written)
}
