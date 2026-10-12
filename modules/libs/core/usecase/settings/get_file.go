package settings

// GetFile reads the file as its person wrote it, and where it stands.
type GetFile struct {
	read func() (written, path string, err error)
}

func NewGetFile(read func() (written, path string, err error)) *GetFile {
	return &GetFile{read: read}
}

// Execute answers a Current holding the file and where it stands, and nothing
// else.
func (u *GetFile) Execute() (Current, error) {
	written, path, err := u.read()
	if err != nil {
		return Current{}, err
	}
	return Current{Written: written, Path: path}, nil
}
