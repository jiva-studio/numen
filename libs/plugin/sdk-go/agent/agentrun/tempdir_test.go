package agentrun

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCreateTempDirCreatesAndCleansUp(t *testing.T) {
	if badDir, nopCleanup, err := CreateTempDir("bad/pattern"); err == nil || badDir != "" || nopCleanup() != nil {
		t.Fatalf("CreateTempDir with path separator = (%q, %v), want empty dir and error", badDir, err)
	}

	dir, cleanup, err := CreateTempDir("numen-test-*")
	if err != nil {
		t.Fatalf("CreateTempDir() failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected directory at %s, err: %v", dir, err)
	}

	lockedSub := filepath.Join(dir, "locked")
	if err := os.Mkdir(lockedSub, 0o700); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(lockedSub, "sample.txt")
	if err := os.WriteFile(testFile, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() != 0 {
		if err := os.Chmod(lockedSub, 0o500); err != nil {
			t.Fatal(err)
		}
		if err := cleanup(); !errors.Is(err, fs.ErrPermission) {
			t.Fatalf("cleanup() on read-only subdirectory = %v, want permission error", err)
		}
		if err := os.Chmod(lockedSub, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	if err := cleanup(); err != nil {
		t.Fatalf("cleanup() failed: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected directory to be deleted, err: %v", err)
	}

	if err := cleanup(); err != nil {
		t.Fatalf("second cleanup() failed: %v", err)
	}
}

func TestRemoveFileReportsTheFailureOfTheRemoval(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "folder")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(nested, "child")
	if err := os.WriteFile(child, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RemoveFile(nested); !errors.Is(err, syscall.ENOTEMPTY) && !errors.Is(err, syscall.EEXIST) {
		t.Errorf("RemoveFile() of a folder that holds a file = %v, want ENOTEMPTY", err)
	}
	if err := RemoveFile(child); err != nil {
		t.Errorf("RemoveFile() of an existing file = %v, want nil", err)
	}
	if err := RemoveFile(filepath.Join(dir, "missing")); err != nil {
		t.Errorf("RemoveFile() of a missing file = %v, want nil", err)
	}
	if err := RemoveFile(""); err != nil {
		t.Errorf("RemoveFile() of an empty path = %v, want nil", err)
	}
}

func TestLinkEntriesLinksEveryEntryExceptTheSkippedOne(t *testing.T) {
	from, into := t.TempDir(), t.TempDir()
	for _, name := range []string{"keep.txt", "skip.txt", "zed.txt"} {
		if err := os.WriteFile(filepath.Join(from, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if err := LinkEntries(from, into, "skip.txt"); err != nil {
		t.Fatalf("LinkEntries() failed: %v", err)
	}
	if err := LinkEntries(from, into, "skip.txt"); err != nil {
		t.Fatalf("LinkEntries() on linked entries failed: %v", err)
	}

	for _, name := range []string{"keep.txt", "zed.txt"} {
		if _, err := os.Lstat(filepath.Join(into, name)); err != nil {
			t.Errorf("%s not linked: %v", name, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(into, "skip.txt")); err == nil {
		t.Error("skip.txt linked")
	}
}

func TestLinkEntriesIgnoresAMissingSource(t *testing.T) {
	if err := LinkEntries(filepath.Join(t.TempDir(), "missing"), t.TempDir(), ""); err != nil {
		t.Errorf("LinkEntries() = %v, want nil", err)
	}
}

func TestLinkEntriesReturnsTheFailureOfALink(t *testing.T) {
	from := t.TempDir()
	entryFile := filepath.Join(from, "entry.txt")
	if err := os.WriteFile(entryFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LinkEntries(entryFile, t.TempDir(), ""); !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf("LinkEntries() from regular file = %v, want ENOTDIR", err)
	}

	if err := LinkEntries(from, filepath.Join(t.TempDir(), "missing"), ""); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("LinkEntries() = %v, want ErrNotExist", err)
	}
}
