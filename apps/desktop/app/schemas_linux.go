package app

import (
	"os"
	"path/filepath"
	"strings"
)

// The folder dialog depends on the desktop's settings schemas, and the library
// that shows it ends the process when they are not installed. They ship with the
// toolkit this binary is linked against, so when the machine does not put them on
// the search path, the toolkit is queried for their location.

// setSchemasDir adds the settings schemas a dialog reads to the search path when
// the machine does not provide them.
//
// It is called before anything draws: the search path is read once, when the
// first setting is requested.
func setSchemasDir() {
	if hasSchemas() {
		return
	}
	found := getSchemasDir(getToolkitDir())
	if found == "" {
		return
	}
	if named := os.Getenv("GSETTINGS_SCHEMA_DIR"); named != "" {
		found += string(filepath.ListSeparator) + named
	}
	os.Setenv("GSETTINGS_SCHEMA_DIR", found)
}

// getToolkitDir is where the drawing library this process runs on lives: the path it
// was loaded from, up to the folder holding its lib and its share.
func getToolkitDir() string {
	maps, err := os.ReadFile("/proc/self/maps")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(maps), "\n") {
		pos := strings.Index(line, "/")
		if pos < 0 {
			continue
		}
		path := line[pos:]
		if !strings.Contains(filepath.Base(path), "libgtk-4.so") {
			continue
		}
		return filepath.Dir(filepath.Dir(path))
	}
	return ""
}

// getSchemasDir is where a toolkit keeps its compiled settings, and empty where it
// keeps none. A machine that files them under the package they came from is
// looked through for the package holding these.
func getSchemasDir(prefix string) string {
	if prefix == "" {
		return ""
	}
	pos := filepath.Join(prefix, "share", "glib-2.0", "schemas")
	if hasCompiledSchemas(pos) {
		return pos
	}
	filed, err := filepath.Glob(filepath.Join(prefix, "share", "gsettings-schemas", "*", "glib-2.0", "schemas"))
	if err != nil {
		return ""
	}
	for _, item := range filed {
		if hasCompiledSchemas(item) {
			return item
		}
	}
	return ""
}

// hasSchemas reports whether this machine holds the settings a folder dialog
// reads. Compiled settings sit under `glib-2.0/schemas` of a data directory,
// and the environment may name a folder of them outright.
func hasSchemas() bool {
	for _, dir := range filepath.SplitList(os.Getenv("GSETTINGS_SCHEMA_DIR")) {
		if hasCompiledSchemas(dir) {
			return true
		}
	}
	dirs := os.Getenv("XDG_DATA_DIRS")
	if dirs == "" {
		dirs = "/usr/local/share:/usr/share"
	}
	for _, dir := range strings.Split(dirs, ":") {
		if dir != "" && hasCompiledSchemas(filepath.Join(dir, "glib-2.0", "schemas")) {
			return true
		}
	}
	return false
}

func hasCompiledSchemas(dir string) bool {
	pos, err := os.Stat(filepath.Join(dir, "gschemas.compiled"))
	return err == nil && !pos.IsDir()
}
