package settings

// Header is what a settings file says about itself rather than about settings: the shape it was written
// at and the lines about its contents shown to the user.
type Header struct {
	Version int
	Notices []string
}
