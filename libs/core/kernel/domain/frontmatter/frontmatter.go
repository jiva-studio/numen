// Package frontmatter holds the values of a read of the frontmatter of notes as the index stores it.
package frontmatter

import (
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
)

// ErrInvalidQuery is a query over the limits of a read.
var ErrInvalidQuery = errors.New("invalid frontmatter query")

// Query names the notes a read returns: the notes at Paths, or every note of the vault when Paths is
// empty, ordered by path, the first Limit after the path After, with the values of Keys, or of every
// key when Keys is empty.
type Query struct {
	Paths []string
	Keys  []string
	After string
	Limit int
}

// Note is the frontmatter of one note: the source the index keeps it as, and the values of the keys
// read, as JSON decodes them.
type Note struct {
	SourceID filechange.SourceID
	Path     string
	Values   map[string]any
}

// Page is one page of a read: its notes, the path the next page starts after, empty when this page
// is the last, and the cursor of the file change feed taken before the page was read.
type Page struct {
	Notes  []Note
	After  string
	Cursor filechange.Cursor
}
