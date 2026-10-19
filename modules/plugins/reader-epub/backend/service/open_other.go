//go:build !windows

package service

import "os"

// openShared opens path for reading. Other processes may delete or rename it while it
// is open.
func openShared(path string) (*os.File, error) {
	return os.Open(path)
}
