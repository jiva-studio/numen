// Package testsupport gives tests the things they all need and none of them
// should spell out.
package testsupport

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// VaultDir returns the fixture vault every test scans.
//
// It walks up from this file until it finds the repository, so that a test says
// what it wants instead of counting parent directories — a chain of `../` is
// both unreadable and wrong the moment a package moves.
func VaultDir(t *testing.T) string {
	t.Helper()

	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test support package")
	}

	dir := filepath.Dir(self)
	for {
		candidate := filepath.Join(dir, "tests", "vault")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("fixture vault not found above " + filepath.Dir(self))
		}
		dir = parent
	}
}

// CopyVault returns a writable copy of the fixture, for tests that change what
// they scan. The fixture itself must stay exactly as committed.
func CopyVault(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	if err := os.CopyFS(dst, os.DirFS(VaultDir(t))); err != nil {
		t.Fatal(err)
	}
	return dst
}
