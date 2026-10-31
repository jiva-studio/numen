package agentrun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTempDirCreatesAndCleansUp(t *testing.T) {
	dir, cleanup, err := CreateTempDir("numen-test-*")
	if err != nil {
		t.Fatalf("CreateTempDir() failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected directory at %s, err: %v", dir, err)
	}

	testFile := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(testFile, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}

	cleanup()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected directory to be deleted, err: %v", err)
	}

	// Calling cleanup again does not error or panic.
	cleanup()
}

func TestLinkEntriesLinksEveryEntryExceptTheSkippedOne(t *testing.T) {
	from, into := t.TempDir(), t.TempDir()
	for _, name := range []string{"keep.txt", "skip.txt"} {
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

	if _, err := os.Lstat(filepath.Join(into, "keep.txt")); err != nil {
		t.Errorf("keep.txt not linked: %v", err)
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
	if err := os.WriteFile(filepath.Join(from, "entry.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := LinkEntries(from, filepath.Join(t.TempDir(), "missing"), ""); err == nil {
		t.Error("LinkEntries() = nil, want the link failure")
	}
}
