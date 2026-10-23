//go:build unix

package vaultreader

import (
	"os"
	"syscall"
)

// openReadable opens a file for reading without blocking on special files.
func openReadable(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}

// openShared opens a file for reading; an open file can always be renamed or
// deleted here.
func openShared(path string) (*os.File, error) { return openReadable(path) }
