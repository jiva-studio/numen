// Package pdf turns the bytes of a PDF file into the text it carries and the
// places it names. It reads no file, asks no clock and holds no database, and
// the same bytes give the same text at the same offsets, which is what lets a
// chunk keep an offset and not the text.
//
// It does hold one thing with a lifetime: a pool of WebAssembly workers,
// started when the first document is opened and kept for the life of the
// process.
//
// A document is read in the order it is paginated: every page's text layer, in
// order, as one stream. Everything the document names — an outline entry, a
// printed page — is an offset into that stream.
//
// A page whose text layer says nothing is an ordinary page. A document that
// says nothing at all is a scan, and it is read to an empty text rather than
// refused: what to do about a scan is decided elsewhere. An error here means
// the file is not a PDF.
package pdf

import (
	"errors"
	"sort"
	"strings"
)

// A Book is one PDF file, read.
type Book struct {
	// Title is the name the document's metadata gives it, and is empty for the
	// many that carry none.
	Title string

	// Text is every page's text layer, in order, as one stream.
	Text string

	// Pages are the pages of the document, ascending by offset. Unlike a book
	// made for a screen, a PDF always has them.
	Pages []Page

	// Parts are what the outline names, ascending by offset. Structure says
	// where they came from.
	Parts []Part

	// Structure names the tier that produced Parts.
	Structure Structure
}

// A Part is a named division of the document.
type Part struct {
	Title string
	// Offset is where the named text begins in the document's Text.
	Offset int
	// Level is the depth of the outline entry a part came from, counting from
	// one. The depth an outline chose carries no meaning of its own.
	Level int
}

// A Page is one page of the document, at the offset where its text begins.
//
// It carries no name. A document that numbers its pages and a viewer that opens
// them count differently, and two numbers for one page is a way to be wrong
// about which page a person is looking at.
type Page struct {
	Offset int
}

// A Structure is the tier that answered when the document was asked what its
// parts are. Each tier is tried only when the one before it yields nothing.
type Structure string

const (
	// FromOutline: the document's own outline named its places.
	FromOutline Structure = "outline"
	// FromNothing: the document names no parts. This is an ordinary result,
	// and the usual one: a text layer carries the size of its type and not the
	// meaning of it, so there is no second tier to fall to.
	FromNothing Structure = "none"
)

// A document that names one part names its cover or its own title. Structure
// begins at two.
const minimumParts = 2

// What one document may spend.
//
// A page of prose is a few thousand characters, so the bound on text is a
// library's worth of them and is reached only by a file built to reach it. The
// bound on pages is what a reader can be shown before the offsets stop meaning
// anything to anybody.
//
// The bound on pixels is what one page is drawn at, whatever its page box says.
// A page of A2 at 300 dots to the inch is inside it.
const (
	mostPages  = 20_000
	mostText   = 256 << 20
	mostPixels = 64 << 20
)

// pointsPerInch is the page's own unit against the inch a resolution is given
// in.
const pointsPerInch = 72

// The ways a file can fail to be a document.
var (
	ErrNotPDF    = errors.New("these bytes do not open as a PDF")
	ErrEncrypted = errors.New("pdf: the file is encrypted")
)

// Read extracts one document from the bytes of a PDF file.
func Read(raw []byte) (*Book, error) {
	doc, err := open(raw)
	if err != nil {
		return nil, err
	}
	defer doc.close()

	book := &Book{Title: doc.title()}

	var text strings.Builder
	pages := min(doc.pages, mostPages)
	starts := make([]int, 0, pages)
	for i := 0; i < pages; i++ {
		if text.Len() >= mostText {
			break
		}
		starts = append(starts, text.Len())
		book.Pages = append(book.Pages, Page{Offset: text.Len()})

		page := doc.text(i)
		if page == "" {
			continue
		}
		text.WriteString(page)
		// Every page ends a line, so that the last word of one page and the
		// first of the next are two words.
		if !strings.HasSuffix(page, "\n") {
			text.WriteString("\n")
		}
	}
	book.Text = text.String()

	if named := doc.outline(starts); len(named) >= minimumParts {
		book.Parts, book.Structure = named, FromOutline
	} else {
		book.Structure = FromNothing
	}
	sort.SliceStable(book.Parts, func(i, j int) bool {
		return book.Parts[i].Offset < book.Parts[j].Offset
	})
	return book, nil
}

// A Location is where an offset in the text falls.
type Location struct {
	// Part is the nearest name at or before the offset, and empty when the
	// offset precedes every name.
	Part string
	// PartOffset is where that part begins.
	PartOffset int
	// Page is where the page the offset falls on stands in the file, counted from
	// the first. It is known for every page, and it is the one thing a page is
	// called.
	Page int
}

// Locate answers where one offset in the document's text is.
func (b *Book) Locate(offset int) Location {
	var at Location
	if i := getPrecedingIndex(len(b.Parts), offset, func(i int) int { return b.Parts[i].Offset }); i >= 0 {
		at.Part, at.PartOffset = b.Parts[i].Title, b.Parts[i].Offset
	}
	if i := getPrecedingIndex(len(b.Pages), offset, func(i int) int { return b.Pages[i].Offset }); i >= 0 {
		at.Page = i
	}
	return at
}

// getPrecedingIndex is the index of the last item at or before an offset, and -1 when
// there is none. The items are ascending by offset.
func getPrecedingIndex(n, offset int, offsetOf func(int) int) int {
	return sort.Search(n, func(i int) bool { return offsetOf(i) > offset }) - 1
}

// tidy is one line of text: the words of a name, single-spaced.
func tidy(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
