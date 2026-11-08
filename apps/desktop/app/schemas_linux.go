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

func setSchemasDir() error {
	if hasSchemas() {
		return nil
	}
	found := getSchemasDir(getToolkitDir())
	if found == "" {
		return nil
	}
	if named := os.Getenv("GSETTINGS_SCHEMA_DIR"); named != "" {
		found += string(filepath.ListSeparator) + named
	}
	return os.Setenv("GSETTINGS_SCHEMA_DIR", found)
}

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

func getSchemasDir(prefix string) string {
	if prefix == "" {
		return ""
	}
	pos := filepath.Join(prefix, "share", "glib-2.0", "schemas")
	if hasCompiledSchemas(pos) {
		return pos
	}
	filed, err := filepath.Glob(filepath.Join(prefix, "share", "gsettings-schemas", "*", "glib-2.0", "schemas"))
	// gomutants:disable-next-line
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

const defaultDataDirs = "/usr/local/share:/usr/share"

func hasSchemas() bool {
	return hasSchemasWithDefault(defaultDataDirs)
}

func hasSchemasWithDefault(fallbackDirs string) bool {
	for _, dir := range filepath.SplitList(os.Getenv("GSETTINGS_SCHEMA_DIR")) {
		if hasCompiledSchemas(dir) {
			return true
		}
	}
	dirs := os.Getenv("XDG_DATA_DIRS")
	if dirs == "" {
		dirs = fallbackDirs
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
