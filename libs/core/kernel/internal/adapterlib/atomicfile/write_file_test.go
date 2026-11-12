package atomicfile_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/atomicfile"
)

func TestWriteFileCreatesAndReplacesWithTheMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	for _, content := range []string{"first\n", "second, and longer\n"} {
		if err := atomicfile.WriteFile(context.Background(), path, strings.NewReader(content), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != content {
			t.Fatalf("file holds %q, want %q", got, content)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 && !isWindows() {
		t.Fatalf("mode %v is open to others", info.Mode())
	}
}

func TestWriteFileKeepsTheOldFileWhenTheSourceFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	broken := errors.New("the disk went away")
	source := io.MultiReader(strings.NewReader("half of the new con"), iotest.ErrReader(broken))
	if err := atomicfile.WriteFile(context.Background(), path, source, 0o600); !errors.Is(err, broken) {
		t.Fatalf("error %v, want %v", err, broken)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old\n" {
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

func TestWriteFileLeavesNothingWhenTheSourceFailsFirst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := atomicfile.WriteFile(context.Background(), path, iotest.ErrReader(errors.New("unreadable")), 0o600); err == nil {
		t.Fatal("a failed read was reported as success")
	}
	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("a failed first write left %d entries", len(left))
	}
}

func TestWriteFileReportsAMissingFolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent", "state.json")
	if err := atomicfile.WriteFile(context.Background(), path, strings.NewReader("x"), 0o600); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error %v, want a missing folder", err)
	}
}

func isWindows() bool { return filepath.Separator == '\\' }
