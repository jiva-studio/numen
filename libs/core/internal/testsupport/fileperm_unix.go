//go:build unix

package testsupport

import (
	"os"
	"path/filepath"
	"testing"
)

// Shut makes a file unopenable and restores access when the test ends. On unix
// this changes the file's permissions.
func Shut(tb testing.TB, path string) {
	tb.Helper()
	if os.Geteuid() == 0 {
		tb.Skip("root opens a file whatever its permissions say")
	}
	if err := os.Chmod(path, 0); err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := os.Chmod(path, 0o644); err != nil {
			tb.Error(err)
		}
	})
}

// MakeUnwritable makes a file impossible to save over and restores it when the
// test ends. A save renames over the name, so the folder containing the name is
// what blocks it.
func MakeUnwritable(tb testing.TB, path string) {
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
	tb.Cleanup(func() {
		if err := os.Chmod(dir, info.Mode().Perm()); err != nil {
			tb.Error(err)
		}
	})
}
