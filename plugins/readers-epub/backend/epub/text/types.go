package text

import "github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/opf"

// A Document is one document of the spine.
type Document struct {
	// Path is the document's name inside the archive.
	Path string
	// Offset is where the document's text begins in the book's Text.
	Offset int
	// Length is how many bytes of Text the document contributed.
	Length int
	// IsLinear is false for a document the spine sets apart from the reading
	// order: a note, an appendix, the back of a plate. Its text is in Text all
	// the same.
	IsLinear bool
	Layout   opf.Layout
}

// A Part is a named division of the book.
type Part struct {
	Title string
	// Offset is where the named text begins in the book's Text.
	Offset int
	// Level is the depth of the heading a part came from, and 0 for a part that
	// came from the navigation document.
	Level int
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
