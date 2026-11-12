package testsupport

import (
	"path/filepath"
	"testing"
)

// TempDir is a temporary folder at the path the application reports for it. A
// vault root is read through its symlinks, and on macOS a temporary folder
func TempDir(tb testing.TB) string {
	tb.Helper()
	dir, err := filepath.EvalSymlinks(tb.TempDir())
	if err != nil {
		tb.Fatal(err)
	}
	return dir
}
