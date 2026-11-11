package vaultfixture

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// VaultDir returns the fixture vault that every test scans.
//
// It walks up from this file until it finds the repository root, so a test
// works unchanged when its package moves.
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
	dst := testsupport.TempDir(t)
	if err := os.CopyFS(dst, os.DirFS(VaultDir(t))); err != nil {
		t.Fatal(err)
	}
	return dst
}

// NewVault writes a vault with the given notes and gives it an identity, for
// tests that need a second vault whose content is nothing like the fixture's.
func NewVault(tb testing.TB, notes map[string]string) domain.Vault {
	tb.Helper()
	root := testsupport.TempDir(tb)
	for name, body := range notes {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	cfg, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, testsupport.GetEpoch())
	if err != nil {
		tb.Fatal(err)
	}
	return domain.Vault{ID: domain.VaultID(cfg.ID), Name: filepath.Base(root), Path: root}
}

// WriteBook writes a book at path inside root, for a test about a vault that
// holds more than notes. The path is slashed and relative to the vault.
func WriteBook(tb testing.TB, root, path string) {
	tb.Helper()
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(target, testsupport.Book(tb), 0o644); err != nil {
		tb.Fatal(err)
	}
}
