package settings

// WriteFile replaces the file whole. A file left standing at anything but the
type WriteFile struct {
	write func(content string, baseline *string) error
}

func NewWriteFile(write func(content string, baseline *string) error) *WriteFile {
	return &WriteFile{write: write}
}

func (u *WriteFile) Execute(content string, baseline *string) error {
	return u.write(content, baseline)
}
