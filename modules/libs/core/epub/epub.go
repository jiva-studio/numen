// Package epub turns the bytes of an EPUB file into the text a book carries and
// the places it names. It is pure: no filesystem, no clock, no database. The
// same bytes give the same text at the same offsets, which is what lets a chunk
// keep an offset and not the text.
//
// A book is read in the order it declares: the container names a package
// document, the package's spine names its documents, and their text is
// concatenated into one stream as far as the bounds a book is read within reach.
// Everything a book names — a navigation entry, a
// heading, a printed page — is an offset into that stream.
//
// A book that names nothing is still read. An error here means the file is not a
// book at all.
package epub

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// A Book is one EPUB file, read.
type Book struct {
	// Title is the name the package document gives the book.
	Title string

	// Text is the spine documents the bounds admit, in reading order, as one
	// stream.
	Text string

	// Documents are the spine documents and where each begins in Text.
	Documents []Document

	// Parts are what the book names, ascending by offset. Tier says where they
	// came from.
	Parts []Part

	// Pages are the pages of the printed book this file was made from,
	// ascending by offset. Most books carry none.
	Pages []Page

	// Tier names the tier that produced Parts.
	Tier Structure
}

// A Document is one document of the spine.
type Document struct {
	// Path is the document's name inside the archive.
	Path string
	// Offset is where the document's text begins in the book's Text.
	Offset int
	// Length is how many bytes of Text the document contributed.
	Length int
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

// A Structure is the tier that answered when the book was asked what its parts
// are. Each tier is tried only when the one before it yields nothing.
type Structure string

const (
	// FromNavigation: the book's own navigation document named its places.
	FromNavigation Structure = "navigation"
	// FromHeadings: the markup carried headings.
	FromHeadings Structure = "headings"
	// FromNothing: the book names no parts. This is an ordinary result.
	FromNothing Structure = "none"
)

// A book that names one part names its cover or its own title. Structure begins
// at two.
const minimumParts = 2

// The three ways a file can fail to be a book.
var (
	ErrNotArchive  = errors.New("epub: not a zip archive")
	ErrNoContainer = errors.New("epub: no META-INF/container.xml")
	ErrNoPackage   = errors.New("epub: no package document")
)

// Read extracts one book from the bytes of an EPUB file.
func Read(raw []byte) (*Book, error) {
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotArchive, err)
	}
	files := archiveIndex(archive)

	opfPath, err := packagePath(files)
	if err != nil {
		return nil, err
	}
	pkg, err := readPackage(files, opfPath)
	if err != nil {
		return nil, err
	}

	text := newExtractor()
	book := &Book{Title: pkg.title}
	left := int64(mostPerBook)
	for _, docPath := range pkg.spine {
		if left <= 0 {
			break
		}
		markup, ok := within(files[docPath], min(int64(mostPerDocument), left))
		if !ok {
			// A manifest may name a file the archive does not hold, and one it
			// holds may be larger than a chapter can be.
			continue
		}
		left -= int64(len(markup))
		book.Documents = append(book.Documents, text.document(docPath, markup))
	}
	book.Text = string(text.out)

	named := navigationParts(files, pkg, text)
	switch {
	case len(named) >= minimumParts:
		book.Parts, book.Tier = named, FromNavigation
	case len(text.headings) >= minimumParts:
		book.Parts, book.Tier = text.headings, FromHeadings
	default:
		book.Tier = FromNothing
	}
	sort.SliceStable(book.Parts, func(i, j int) bool {
		return book.Parts[i].Offset < book.Parts[j].Offset
	})

	book.Pages = text.pagesAt(pageEntries(files, pkg))
	sort.SliceStable(book.Pages, func(i, j int) bool {
		return book.Pages[i].Offset < book.Pages[j].Offset
	})
	return book, nil
}

// A Location is where an offset in the text falls.
type Location struct {
	// Document is the spine document holding the offset.
	Document string
	// Part is the nearest name at or before the offset, and empty when the
	// offset precedes every name.
	Part string
	// PartOffset is where that part begins.
	PartOffset int
	// Page is the printed page in force, and empty when the book has none.
	Page string
}

// Locate answers where one offset in the book's text is.
func (b *Book) Locate(offset int) Location {
	var at Location
	if i := preceding(len(b.Documents), offset, func(i int) int { return b.Documents[i].Offset }); i >= 0 {
		at.Document = b.Documents[i].Path
	}
	if i := preceding(len(b.Parts), offset, func(i int) int { return b.Parts[i].Offset }); i >= 0 {
		at.Part, at.PartOffset = b.Parts[i].Title, b.Parts[i].Offset
	}
	if i := preceding(len(b.Pages), offset, func(i int) int { return b.Pages[i].Offset }); i >= 0 {
		at.Page = b.Pages[i].Label
	}
	return at
}

// preceding is the index of the last item at or before an offset, and -1 when
// there is none. The items are ascending by offset.
func preceding(n, offset int, offsetOf func(int) int) int {
	return sort.Search(n, func(i int) bool { return offsetOf(i) > offset }) - 1
}

// tidy is one line of text: the words of a name, single-spaced.
func tidy(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
