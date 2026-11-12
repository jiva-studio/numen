package atomicfile_test

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/jiva-studio/numen/libs/atomicfile"
)

func openRoot(t *testing.T, dir string) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return root
}

func TestReplaceWritesTheContentAndReportsItsSize(t *testing.T) {
	dir := t.TempDir()
	root := openRoot(t, dir)
	stat, err := atomicfile.Replace(t.Context(), root, "notes.txt", strings.NewReader("twelve bytes"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dir + "/notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "twelve bytes" || stat.Size != 12 {
		t.Fatalf("file holds %q and reports %d bytes", got, stat.Size)
	}
	info, err := os.Stat(dir + "/notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(stat.ModTime) {
		t.Fatalf("modification time %v, reported %v", info.ModTime(), stat.ModTime)
	}
	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 {
		t.Fatalf("folder holds %d entries, want only the file", len(left))
	}
}

func TestReplaceKeepsTheOldFileWhenTheSourceFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/notes.txt", []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	broken := errors.New("the disk went away")
	source := io.MultiReader(strings.NewReader("half"), iotest.ErrReader(broken))
	if _, err := atomicfile.Replace(t.Context(), openRoot(t, dir), "notes.txt", source, 0o600); !errors.Is(err, broken) {
		t.Fatalf("error %v, want %v", err, broken)
	}
	got, err := os.ReadFile(dir + "/notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("file holds %q after a failed write", got)
	}
	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 {
		t.Fatalf("folder holds %d entries after a failed write, want only the file", len(left))
	}
}

func TestReplaceWritesATargetWhoseNameFillsTheLimit(t *testing.T) {
	dir := t.TempDir()
	name := strings.Repeat("é", 127) + ".md"
	if _, err := atomicfile.Replace(t.Context(), openRoot(t, dir), name, strings.NewReader("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir + "/" + name); err != nil {
		t.Fatal(err)
	}
}

func TestReplaceReportsAMissingFolder(t *testing.T) {
	root := openRoot(t, t.TempDir())
	if _, err := atomicfile.Replace(t.Context(), root, "absent/notes.txt", strings.NewReader("x"), 0o644); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error %v, want a missing folder", err)
	}
}

func TestSyncDirectoryFlushesAFolderAndAcceptsAMissingOne(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(dir+"/inner", 0o755); err != nil {
		t.Fatal(err)
	}
	root := openRoot(t, dir)
	if err := atomicfile.SyncDirectory(root, "inner"); err != nil {
		t.Fatalf("flush of a folder: %v", err)
	}
	if err := atomicfile.SyncDirectory(root, "absent"); err != nil {
		t.Fatalf("flush of a missing folder: %v", err)
	}
}
