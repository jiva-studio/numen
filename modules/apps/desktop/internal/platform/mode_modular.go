package platform

var isMonolith bool

// BuildMode is the desktop build variant.
const BuildMode = "modular"

// IsMonolith reports whether plugins are compiled directly into the binary.
func IsMonolith() bool {
	return isMonolith
}

// SetMonolith sets whether plugins are compiled directly into the binary.
func SetMonolith(v bool) {
	isMonolith = v
}
