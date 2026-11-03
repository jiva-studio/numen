package settings

// Write puts settings into the file. A value the settings cannot be read out of
// again is refused, and the file is left as it was.
type Write struct {
	write func(settings []Setting, applied func()) error
}

func NewWrite(write func(settings []Setting, applied func()) error) *Write {
	return &Write{write: write}
}

// Execute writes the settings. applied, when given, runs once every plugin the
// settings name has been given its new values, which can be after Execute returns.
func (u *Write) Execute(settings []Setting, applied func()) error {
	return u.write(settings, applied)
}
