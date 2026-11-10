package theme

// The words a color scheme is written as.
const (
	SchemeSystem = "system"
	SchemeLight  = "light"
	SchemeDark   = "dark"
)

// ParseColorScheme returns the scheme a word names. A word that names none is the machine's own
// choice.
func ParseColorScheme(word string) ColorScheme {
	switch word {
	case SchemeLight:
		return Light
	case SchemeDark:
		return Dark
	}
	return System
}

// GetName is the word the scheme is written as.
func (s ColorScheme) GetName() string {
	switch s {
	case Light:
		return SchemeLight
	case Dark:
		return SchemeDark
	default:
		return SchemeSystem
	}
}
