//go:build !windows && !darwin

package filesystem

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// documents is what the desktop was told to call this folder.
//
// `xdg-user-dirs` writes the answer into a file of shell assignments, and the
// environment overrides it for one session. Both name the folder in whatever
// language the desktop was set up in, which is why neither is guessed at.
func documents() (string, error) {
	if named := os.Getenv("XDG_DOCUMENTS_DIR"); named != "" {
		return expand(named)
	}
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		config = filepath.Join(home, ".config")
	}
	raw, err := os.ReadFile(filepath.Join(config, "user-dirs.dirs"))
	if err != nil {
		return "", err
	}
	for line := range strings.Lines(string(raw)) {
		name, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found || name != "XDG_DOCUMENTS_DIR" {
			continue
		}
		return expand(value)
	}
	return "", os.ErrNotExist
}

// expand is one assignment's value as a path: the quotes come off, and the one
// variable such a file is written with is filled in.
func expand(value string) (string, error) {
	if unquoted, err := strconv.Unquote(value); err == nil {
		value = unquoted
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	value = strings.ReplaceAll(value, "$HOME", home)
	return strings.ReplaceAll(value, "${HOME}", home), nil
}
