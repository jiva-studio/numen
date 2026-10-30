package derivedfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

// The rollback of an append truncates only what that append wrote. A second
// writer landing between opening the file and the write moves where this
// writer's bytes begin, so the original file length is not the rollback point.
func TestAnAppendIsTakenBackToWhereItLanded(t *testing.T) {
	dir := t.TempDir()
	pos := filepath.Join(dir, "run.txt")
	if err := os.WriteFile(pos, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, root)

	file, err := root.OpenFile("run.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	other, err := os.OpenFile(pos, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		testsupport.Close(t, file)
		t.Fatal(err)
	}
	if _, err := other.Write([]byte("two\n")); err != nil {
		t.Fatal(err)
	}
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}

	wrote, err := file.Write([]byte("three\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := closeAndTruncate(root, file, "run.txt", wrote); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(pos)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "one\ntwo\n" {
		t.Errorf("the file holds %q, want both lines that landed whole", got)
	}
}

// A write that landed nothing has nothing to take back.
func TestAnAppendThatLandedNothingSplitsNothing(t *testing.T) {
	dir := t.TempDir()
	pos := filepath.Join(dir, "run.txt")
	if err := os.WriteFile(pos, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, root)
	file, err := root.OpenFile("run.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := closeAndTruncate(root, file, "run.txt", 0); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(pos); err != nil || string(got) != "one\n" {
		t.Errorf("the file holds %q, want what stood there: %v", got, err)
	}
}
