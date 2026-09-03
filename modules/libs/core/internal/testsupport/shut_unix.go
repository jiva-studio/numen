//go:build unix

package testsupport

import (
	"os"
	"path/filepath"
	"testing"
)

// Shut makes a file nobody may open, and opens it again when the test ends.
// Here that is the file's own permissions.
func Shut(tb testing.TB, path string) {
	tb.Helper()
	if os.Geteuid() == 0 {
		tb.Skip("root opens a file whatever its permissions say")
	}
	if err := os.Chmod(path, 0); err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { os.Chmod(path, 0o644) })
}

// Unwritable makes a file nothing may save over, and lets it be saved again
// when the test ends. A save lands by renaming over the name, so what refuses
// it is the folder the name is in.
func Unwritable(tb testing.TB, path string) {
	tb.Helper()
	if os.Geteuid() == 0 {
		tb.Skip("root writes a folder whatever its permissions say")
	}
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		tb.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { os.Chmod(dir, info.Mode().Perm()) })
}
