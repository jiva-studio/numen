//go:build !unix

package filesystem

import "os"

// openReadable opens a file for reading.
func openReadable(path string) (*os.File, error) {
	return os.Open(path)
}
