//go:build !linux

package app

// setSchemasDir leaves the search path as it is. Only the toolkit on Linux is
// built on the settings a dialog reads.
func setSchemasDir() {}

// hasSchemas reports whether this machine holds the settings a folder dialog
// reads.
func hasSchemas() bool { return true }
