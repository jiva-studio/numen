package filesystem

import (
	"os"
	"path/filepath"
)

// Documents is where this person keeps their documents, as the system they are
// on answers it.
//
// The answer is asked for rather than assembled: the folder is renamed, moved
// and localised, and a path built out of a home directory and a word is right
// only until somebody does any of that. Where the system says nothing, the
// name it would have said is what is used.
func Documents() (string, error) {
	if named, err := documents(); err == nil && filepath.IsAbs(named) {
		return filepath.Clean(named), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents"), nil
}
