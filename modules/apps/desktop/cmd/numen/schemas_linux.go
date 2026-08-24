package main

import (
	"os"
	"path/filepath"
	"strings"
)

// The folder picker is built on the desktop's settings, and the library that
// puts it up ends the process where they are not installed. They ship with the
// toolkit this binary is linked against, so where the machine does not put them
// on the search path, the toolkit itself is asked where it keeps them.

// findSchemas puts the settings a picker reads on the search path, where this
// machine has not.
//
// It is called before anything draws: the search path is read once, the first
// time something asks for a setting.
func findSchemas() {
	if settled() {
		return
	}
	found := schemasOf(toolkit())
	if found == "" {
		return
	}
	if named := os.Getenv("GSETTINGS_SCHEMA_DIR"); named != "" {
		found += string(filepath.ListSeparator) + named
	}
	os.Setenv("GSETTINGS_SCHEMA_DIR", found)
}

// toolkit is where the drawing library this process runs on lives: the path it
// was loaded from, up to the folder holding its lib and its share.
func toolkit() string {
	maps, err := os.ReadFile("/proc/self/maps")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(maps), "\n") {
		at := strings.Index(line, "/")
		if at < 0 {
			continue
		}
		path := line[at:]
		if !strings.Contains(filepath.Base(path), "libgtk-4.so") {
			continue
		}
		return filepath.Dir(filepath.Dir(path))
	}
	return ""
}

// schemasOf is where a toolkit keeps its compiled settings, and empty where it
// keeps none. A machine that files them under the package they came from is
// looked through for the package holding these.
func schemasOf(prefix string) string {
	if prefix == "" {
		return ""
	}
	at := filepath.Join(prefix, "share", "glib-2.0", "schemas")
	if compiled(at) {
		return at
	}
	filed, err := filepath.Glob(filepath.Join(prefix, "share", "gsettings-schemas", "*", "glib-2.0", "schemas"))
	if err != nil {
		return ""
	}
	for _, at := range filed {
		if compiled(at) {
			return at
		}
	}
	return ""
}

// settled reports whether this machine holds the settings a folder picker
// reads. Compiled settings sit under `glib-2.0/schemas` of a data directory,
// and the environment may name a folder of them outright.
func settled() bool {
	for _, dir := range filepath.SplitList(os.Getenv("GSETTINGS_SCHEMA_DIR")) {
		if compiled(dir) {
			return true
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
