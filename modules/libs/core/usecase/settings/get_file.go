package settings

// GetFile reads the file as its person wrote it, and at its current position.
type GetFile struct {
	read func() (written, path string, err error)
}

func NewGetFile(read func() (written, path string, err error)) *GetFile {
	return &GetFile{read: read}
}

// Execute answers a Current holding the file and at its current position, and nothing
// else.
func (u *GetFile) Execute() (Current, error) {
	content, path, err := u.read()
	if err != nil {
		return Current{}, err
	}
	return Current{Written: content, Path: path}, nil
}
