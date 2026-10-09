//go:build unix

package filesystem

import (
	"os"
	"syscall"
)

// openReadable opens a file for reading without blocking on special files.
func openReadable(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
