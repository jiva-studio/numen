package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// The rollback of an append cuts what that append wrote and nothing before it.
// A second writer landing between the opening of the file and the write moves
// where this one's bytes begin, and the length the file had is no longer it.
func TestAnAppendIsTakenBackToWhereItLanded(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "run.txt")
	if err := os.WriteFile(at, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	file, err := root.OpenFile("run.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	other, err := os.OpenFile(at, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		file.Close()
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
	if err := back(root, file, "run.txt", wrote); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "one\ntwo\n" {
		t.Errorf("the file holds %q, want both lines that landed whole", got)
	}
}

// A write that landed nothing has nothing to take back.
func TestAnAppendThatLandedNothingCutsNothing(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "run.txt")
	if err := os.WriteFile(at, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	file, err := root.OpenFile("run.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := back(root, file, "run.txt", 0); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(at); err != nil || string(got) != "one\n" {
		t.Errorf("the file holds %q, want what stood there: %v", got, err)
	}
}
