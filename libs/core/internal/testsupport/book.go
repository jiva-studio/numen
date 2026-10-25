package testsupport

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteBook writes a book at path inside root, for a test about a vault that
// holds more than notes. The path is slashed and relative to the vault.
func WriteBook(tb testing.TB, root, path string) {
	tb.Helper()
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(target, Book(tb), 0o644); err != nil {
		tb.Fatal(err)
	}
}

// Book is the bytes of a book with one chapter in it, in the form Documents
// reads. The fixture vault stays text a reader can check.
func Book(tb testing.TB) []byte {
	tb.Helper()
	return NewBook("A book carries text and no title of its own in the graph.")
}
