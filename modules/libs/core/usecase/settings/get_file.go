package settings

// GetFile reads the file as its person wrote it, and where it stands.
type GetFile struct {
	read func() (written, path string, err error)
}

// File is the settings file as its person wrote it, and where it stands.
type File struct {
	Written string
	Path    string
}

func NewGetFile(read func() (written, path string, err error)) GetFile {
	return GetFile{read: read}
}

func (u GetFile) Execute() (File, error) {
	written, path, err := u.read()
	if err != nil {
		return File{}, err
	}
	return File{Written: written, Path: path}, nil
}
