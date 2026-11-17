// Package tempdir creates and removes folders in the temporary folder of the system.
package tempdir

import "os"

// Folders creates folders named after Pattern, as os.MkdirTemp reads it.
type Folders struct {
	Pattern string
}

// Create creates a new folder and returns its path.
func (f Folders) Create() (string, error) {
	return os.MkdirTemp("", f.Pattern)
}

// Remove removes the folder and everything in it.
func (Folders) Remove(dir string) error {
	return os.RemoveAll(dir)
}
