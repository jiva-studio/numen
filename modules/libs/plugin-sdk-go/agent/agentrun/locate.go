package agentrun

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// Locate returns the first of names found on the path, else the first runnable file the places
// match. A place is a glob pattern, and one starting with "~/" stands under the home folder. Empty
// says the program is nowhere.
func Locate(names, places []string) string {
	for _, name := range names {
		if found, err := exec.LookPath(name); err == nil {
			return found
		}
	}
	return findInPlaces(places)
}

// findInPlaces returns the first runnable file the places match, empty where none does. A place
// under the home is skipped where there is no home.
func findInPlaces(places []string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	for _, place := range places {
		if strings.HasPrefix(place, "~/") {
			if home == "" {
				continue
			}
			place = filepath.Join(home, place[2:])
		}
		matches, err := filepath.Glob(place)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if isRunnable(match) {
				return match
			}
		}
	}
	return ""
}

// isRunnable is a regular file with an execute bit, held under the exact name it was looked for by.
func isRunnable(path string) bool {
	about, err := os.Stat(path)
	if err != nil || !about.Mode().IsRegular() || about.Mode().Perm()&0o111 == 0 {
		return false
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return false
	}
	name := filepath.Base(path)
	return slices.ContainsFunc(entries, func(e os.DirEntry) bool { return e.Name() == name })
}
