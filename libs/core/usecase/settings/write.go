package settings

// Write puts settings into the file. A value the settings cannot be read out of
// again is refused, and the file is left as it was.
type Write struct {
	write func(settings []Setting) error
}

func NewWrite(write func(settings []Setting) error) *Write {
	return &Write{write: write}
}

// Execute writes the settings. A plugin is given its new values after Execute returns.
func (u *Write) Execute(settings []Setting) error {
	return u.write(settings)
}
