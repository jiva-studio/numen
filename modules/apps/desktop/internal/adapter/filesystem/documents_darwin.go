package filesystem

import (
	"os"
	"path/filepath"
)

// documents is the folder macOS keeps documents in. The name on disk is always
// this one; what Finder shows is a display name and does not move the folder.
func documents() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents"), nil
}
