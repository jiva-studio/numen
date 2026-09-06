package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// The lock map is never pruned, so what bounds it is what can be a key. A key
// is the folder as the filesystem resolves it, so every spelling of one vault
// root is the one entry — and two writers on one folder spelled two ways are
// held apart, which is what the lock is for.
func TestEverySpellingOfAVaultRootIsTheOneLock(t *testing.T) {
	root := t.TempDir()
	linked := filepath.Join(t.TempDir(), "vault")
	if err := os.Symlink(root, linked); err != nil {
		t.Skipf("this machine does not make symlinks: %v", err)
	}

	want := lockFor(root)
	for _, spelling := range []string{
		root,
		root + string(filepath.Separator),
		filepath.Join(root, "."),
		filepath.Join(root, "notes", ".."),
		linked,
	} {
		if got := lockFor(spelling); got != want {
			t.Errorf("%q is locked under a second lock of its own", spelling)
		}
	}
}
