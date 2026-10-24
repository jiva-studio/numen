package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenSharedLetsTheFileBeReplacedWhileOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	if err := os.WriteFile(path, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := openShared(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	next := filepath.Join(dir, "next.epub")
	if err := os.WriteFile(next, []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(next, path); err != nil {
		t.Fatalf("replace the open file: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("delete the open file: %v", err)
	}
}
