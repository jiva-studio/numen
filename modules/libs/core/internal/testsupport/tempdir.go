package testsupport

import (
	"path/filepath"
	"testing"
)

// TempDir is a temporary folder at the path the application reports for it. A
// vault root is read through its symlinks, and on macOS a temporary folder
// stands behind one.
func TempDir(t testing.TB) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
