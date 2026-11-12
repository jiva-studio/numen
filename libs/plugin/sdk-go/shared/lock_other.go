//go:build !unix && !windows

package shared

import "errors"

func lockFile(string) (func() error, error) {
	return nil, errors.New("this platform has no file lock")
}
