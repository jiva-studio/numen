package settings

// Write puts settings into the file. A value the settings cannot be read out of
type Write struct {
	write func(settings []Setting) error
}

func NewWrite(write func(settings []Setting) error) *Write {
	return &Write{write: write}
}

func (u *Write) Execute(settings []Setting) error {
	return u.write(settings)
}
