package testsupport

import (
	"strings"
	"testing"
)

// Book is the bytes of a book with one chapter in it, in the form Texts
// reads. The fixture vault stays text a reader can check.
func Book(tb testing.TB) []byte {
	tb.Helper()
	return NewBook("A book carries text and no title of its own in the graph.")
}

// bookMark opens the bytes of a book Texts reads. The words follow it, and each line of them is a
// section.
const bookMark = "%BOOK\n"

// NewBook is the bytes of a book Texts reads: the words, one line to a section.
func NewBook(words ...string) []byte {
	return []byte(bookMark + strings.Join(words, "\n"))
}
