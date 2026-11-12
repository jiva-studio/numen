//go:build windows

package vaultreader

import "os"

func openReadableFile(path string) (*os.File, error) {
	return os.Open(path)
}
