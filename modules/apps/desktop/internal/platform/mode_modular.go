//go:build !plugins_monolith

package platform

// BuildMode is the desktop build variant.
const BuildMode = "modular"

// IsMonolith reports whether plugins are compiled directly into the binary.
func IsMonolith() bool {
	return false
}
