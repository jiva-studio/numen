package theme

type ReadTheme struct {
	catalog catalog
}

func NewReadTheme(sources Sources) *ReadTheme {
	return &ReadTheme{catalog: catalog{sources: sources}}
}

// Execute returns the stylesheet the name selects. It returns an empty text, and no error, where no
// theme answers for the name or the user's folder is missing.
func (u *ReadTheme) Execute(name string) (string, error) {
	text, err := u.catalog.read(name)
	if isMissingTheme(err) {
		return "", nil
	}
	return text, err
}
