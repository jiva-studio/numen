package agentrun

import "os"

// CreateTempDir creates a temporary directory matching pattern, and returns its path and a clean-up
// function that deletes it.
func CreateTempDir(pattern string) (string, func(), error) {
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", func() {}, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}
