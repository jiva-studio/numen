package text

import "github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/opf"

// A Document is one document of the spine.
type Document struct {
	Path     string
	Offset   int
	Length   int
	IsLinear bool
	Layout   opf.Layout
	Spine    int
}

// A Part is a named division of the book.
type Part struct {
	Title  string
	Offset int
	Level  int
}

// A Page is a page of the printed book, at the offset where it starts.
type Page struct {
	Label  string
	Offset int
}

// An Entry is one line of a navigation document: a name, and the href it points
// at.
type Entry struct {
	Title string
	Href  string
}
