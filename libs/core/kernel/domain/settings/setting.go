package settings

// Setting is one setting of the file in which this installation is configured,
// with its value.
type Setting struct {
	Path []string
	JSON string
}
