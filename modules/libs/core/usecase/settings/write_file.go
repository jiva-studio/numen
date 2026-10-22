package settings

// WriteFile replaces the file whole. A file left standing at anything but the
type WriteFile struct {
	write func(written string, baseline *string) error
}

func NewWriteFile(write func(written string, baseline *string) error) *WriteFile {
	return &WriteFile{write: write}
}

func (u *WriteFile) Execute(written string, baseline *string) error {
	return u.write(written, baseline)
}
