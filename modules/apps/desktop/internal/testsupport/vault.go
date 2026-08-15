// Package testsupport gives tests the things they all need and none of them
// should spell out.
package testsupport

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
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

// NewVault writes a vault with the given notes and gives it an identity, for
// tests that need a second vault whose content is nothing like the fixture's.
func NewVault(t *testing.T, notes map[string]string) domain.Vault {
	t.Helper()
	root := t.TempDir()
	for name, body := range notes {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return domain.Vault{ID: cfg.ID, Name: filepath.Base(root), Path: root}
}
