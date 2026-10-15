//go:build plugins_monolith

package platform

// BuildMode is the desktop build variant.
const BuildMode = "monolith"

// IsMonolith reports whether plugins are compiled directly into the binary.
func IsMonolith() bool {
	return true
}
