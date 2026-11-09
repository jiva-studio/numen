package vaultwriter_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func placeTrashed(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, path := range paths {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("# Entropy\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEmptyTrashDeletesNestedFoldersAndKeepsTheTrash(t *testing.T) {
	w, root := openWriter(t)
	placeTrashed(t, root, ".trash/Entropy.md", ".trash/physics/Heat.md", ".trash/physics/deep/Work.md", "Kept.md")
	if err := os.MkdirAll(filepath.Join(root, ".trash", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	undeleted, err := w.EmptyTrash(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(undeleted) != 0 {
		t.Errorf("undeleted: %v", undeleted)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".trash"))
	if err != nil {
		t.Fatalf("the trash folder is gone: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("the trash still holds %d entries", len(entries))
	}
	if _, err := os.Stat(filepath.Join(root, "Kept.md")); err != nil {
		t.Errorf("a file outside the trash was touched: %v", err)
	}
}

func TestEmptyTrashWithoutATrashSucceeds(t *testing.T) {
	w, _ := openWriter(t)

	undeleted, err := w.EmptyTrash(t.Context())
	if err != nil || len(undeleted) != 0 {
		t.Errorf("got %v, %v", undeleted, err)
	}
}

func TestEmptyTrashDoesNotFollowALinkOutOfTheTrash(t *testing.T) {
	w, root := openWriter(t)
	outside := t.TempDir()
	placeTrashed(t, outside, "Secret.md")
	if err := os.MkdirAll(filepath.Join(root, ".trash"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".trash", "link")); err != nil {
		t.Skipf("symlinks are not available: %v", err)
	}

	if _, err := w.EmptyTrash(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outside, "Secret.md")); err != nil {
		t.Errorf("a file behind a link was deleted: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".trash", "link")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the link is still in the trash: %v", err)
	}
}

func TestEmptyTrashRemovesATrashThatIsALinkWithoutWalkingIt(t *testing.T) {
	w, root := openWriter(t)
	placeTrashed(t, root, "notes/Kept.md")
	if err := os.Symlink(filepath.Join(root, "notes"), filepath.Join(root, ".trash")); err != nil {
		t.Skipf("symlinks are not available: %v", err)
	}

	undeleted, err := w.EmptyTrash(t.Context())
	if err != nil || len(undeleted) != 0 {
		t.Fatalf("got %v, %v", undeleted, err)
	}
	if _, err := os.Stat(filepath.Join(root, "notes", "Kept.md")); err != nil {
		t.Errorf("a file behind the linked trash was deleted: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".trash")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the trash link is still there: %v", err)
	}
}

func TestEmptyTrashReportsWhatItCannotDeleteAndGoesOn(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission bits do not lock a folder here")
	}
	w, root := openWriter(t)
	placeTrashed(t, root, ".trash/Entropy.md", ".trash/locked/Heat.md", ".trash/physics/Work.md")
	locked := filepath.Join(root, ".trash", "locked")
	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(locked, 0o755); err != nil {
			t.Error(err)
		}
	})

	undeleted, err := w.EmptyTrash(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(undeleted, []string{".trash/locked/Heat.md"}) {
		t.Errorf("undeleted: %v", undeleted)
	}
	for _, target := range []string{".trash/Entropy.md", ".trash/physics"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(target))); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s was not deleted: %v", target, err)
		}
	}
}
