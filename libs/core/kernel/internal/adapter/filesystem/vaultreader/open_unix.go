//go:build unix

package vaultreader

import (
	"os"
	"syscall"
)

func openReadableFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
