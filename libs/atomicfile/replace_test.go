package atomicfile_test

import (
	"crypto/sha256"
	"encoding/hex"
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
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
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
	name := strings.Repeat("é", 126) + ".md"
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

func TestReplaceReportsTheHashOfTheNewContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/notes.txt", []byte("the old content"), 0o600); err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("a line of the new content\n", 10_000)
	stat, err := atomicfile.Replace(t.Context(), openRoot(t, dir), "notes.txt", strings.NewReader(content), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if want := hashOf([]byte(content)); stat.Hash != want {
		t.Fatalf("hash %s, want %s", stat.Hash, want)
	}
	onDisk, err := os.ReadFile(dir + "/notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if stat.Hash != hashOf(onDisk) {
		t.Fatalf("hash %s is not the hash of the file on disk", stat.Hash)
	}
	if stat.Hash == hashOf([]byte("the old content")) {
		t.Fatal("hash is of the old content")
	}
}

func TestReplaceReportsTheHashOfAnEmptyContent(t *testing.T) {
	stat, err := atomicfile.Replace(t.Context(), openRoot(t, t.TempDir()), "empty.txt", strings.NewReader(""), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Hash != hashOf(nil) {
		t.Fatalf("hash %s, want the hash of no bytes", stat.Hash)
	}
}

func TestReplaceReturnsNoStatWhenTheSourceFails(t *testing.T) {
	dir := t.TempDir()
	broken := errors.New("the disk went away")
	source := io.MultiReader(strings.NewReader("half"), iotest.ErrReader(broken))
	stat, err := atomicfile.Replace(t.Context(), openRoot(t, dir), "notes.txt", source, 0o600)
	if !errors.Is(err, broken) {
		t.Fatalf("error %v, want %v", err, broken)
	}
	if stat != (atomicfile.Stat{}) {
		t.Fatalf("stat %+v after a failed write, want none", stat)
	}
}

func hashOf(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func TestReplaceReportsTheHashInTheFormOfHashContent(t *testing.T) {
	stat, err := atomicfile.Replace(t.Context(), openRoot(t, t.TempDir()), "abc.txt", strings.NewReader("abc"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if stat.Hash != want {
		t.Fatalf("hash %s, want %s", stat.Hash, want)
	}
}
