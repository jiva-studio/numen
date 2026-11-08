//go:build unix

package vaultreader

import (
	"os"
	"syscall"
)

func openReadable(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
