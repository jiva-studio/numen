//go:build !unix && !windows

package vaultreader

import "os"

// openReadable opens a file for reading.
func openReadable(path string) (*os.File, error) {
	return os.Open(path)
}
