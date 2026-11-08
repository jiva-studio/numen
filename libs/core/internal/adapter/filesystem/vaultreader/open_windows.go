//go:build windows

package vaultreader

import "os"

func openReadable(path string) (*os.File, error) {
	return os.Open(path)
}
