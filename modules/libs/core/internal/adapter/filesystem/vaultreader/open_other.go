//go:build !unix && !windows

package vaultreader

import "os"

// openReadable opens a file for reading.
func openReadable(path string) (*os.File, error) {
	return os.Open(path)
}

// openShared opens a file for reading.
func openShared(path string) (*os.File, error) { return openReadable(path) }
