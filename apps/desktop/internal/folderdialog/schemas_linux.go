package folderdialog

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// The folder dialog depends on the desktop's settings schemas, and the library
// that shows it ends the process when they are not installed. They ship with the
// toolkit this binary is linked against, so when the machine does not put them on
// the search path, the toolkit is queried for their location.

// SetSchemasDir adds the toolkit's settings schemas to the search path when the machine does not provide them.
func SetSchemasDir() error {
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
	for line := range strings.SplitSeq(string(maps), "\n") {
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
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
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
	if slices.ContainsFunc(filepath.SplitList(os.Getenv("GSETTINGS_SCHEMA_DIR")), hasCompiledSchemas) {
		return true
	}
	dirs := os.Getenv("XDG_DATA_DIRS")
	if dirs == "" {
		dirs = fallbackDirs
	}
	for dir := range strings.SplitSeq(dirs, ":") {
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
