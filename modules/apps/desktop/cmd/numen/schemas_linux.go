package main

import (
	"os"
	"path/filepath"
	"strings"
)

// settled reports whether this machine holds the settings a folder picker reads.
//
// The picker is built by a library that ends the process where they are
// missing, so it is asked for only where they are there. Compiled schemas sit
// under `glib-2.0/schemas` of a data directory, and the environment may name
// one outright.
func settled() bool {
	if named := os.Getenv("GSETTINGS_SCHEMA_DIR"); named != "" {
		for _, dir := range filepath.SplitList(named) {
			if compiled(dir) {
				return true
			}
		}
	}
	dirs := os.Getenv("XDG_DATA_DIRS")
	if dirs == "" {
		dirs = "/usr/local/share:/usr/share"
	}
	for _, dir := range strings.Split(dirs, ":") {
		if dir != "" && compiled(filepath.Join(dir, "glib-2.0", "schemas")) {
			return true
		}
	}
	return false
}

func compiled(dir string) bool {
	at, err := os.Stat(filepath.Join(dir, "gschemas.compiled"))
	return err == nil && !at.IsDir()
}
