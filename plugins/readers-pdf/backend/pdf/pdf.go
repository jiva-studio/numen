// Package pdf converts the bytes of a PDF file into the text they contain and
// the locations they name. It reads no file, uses no clock and holds no
// database, and the same bytes produce the same text at the same offsets, so
// stored offsets stay valid.
//
// It does hold one thing with a lifetime: a pool of WebAssembly workers,
// started when the first document is opened and kept for the life of the
// process.
//
// A document is read in the order it is paginated: every page's text layer, in
// order, as one stream. Everything the document names — an outline entry, a
// printed page — is an offset into that stream.
//
// A page with an empty text layer is an ordinary page. A document with no text
// on any page is a scan, and it yields empty text and not a not-a-PDF error.
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

	// Parts are the outline entries, ascending by offset. Structure records their
	// source.
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
// them count differently, and two numbers for one page risk identifying the
// wrong page.
type Page struct {
	Offset int
}

// A Structure is the tier that supplied the document's parts. Each tier is tried
// only when the one before it yields nothing.
type Structure string

const (
	// FromOutline is the structure when the document's own outline named its places.
	FromOutline Structure = "outline"
	// FromNothing is the structure when the document names no parts. This is an ordinary result,
	// and the usual one: a text layer carries the size of its type and not the
	// meaning of it, so there is no second tier to fall to.
	FromNothing Structure = "none"
)

// A document that names one part names its cover or its own title. Structure
// begins at two.
// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="minimumParts requires at least two parts to constitute document structure"
const minimumParts = 2

// What one document may spend.
//
// A page of prose is a few thousand characters, so the bound on text is a
// library's worth of them and is reached only by a file built to reach it. The
// bound on pages is what a reader can be shown before the offsets stop meaning
// anything to anybody.
//
// The bound on pixels limits the rendered size of one page, regardless of its
// page box. A page of A2 at 300 dots per inch is within it.
const (
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="mostPages limits maximum processed pages in a document"
	mostPages = 20_000
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT,ARITHMETIC_BASE reason="mostText limits maximum extracted text bytes"
	mostText = 256 << 20
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT,ARITHMETIC_BASE reason="mostPixels limits maximum rendered page raster size"
	mostPixels = 64 << 20
)

// pointsPerInch is the page's own unit against the inch a resolution is given
// in.
// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="pointsPerInch defines PDF standard 72 points per inch"
const pointsPerInch = 72

// The ways a file can fail to be a document.
var (
	ErrNotPDF    = errors.New("these bytes do not open as a PDF")
	ErrEncrypted = errors.New("pdf: the file is encrypted")
)

// Read extracts one document from the bytes of a PDF file.
func Read(raw []byte) (*Book, error) {
	doc, err := openBytes(raw)
	if err != nil {
		return nil, err
	}
	return readBook(doc), nil
}

// readBook reads the open document and closes it.
func readBook(doc *document) *Book {
	defer doc.close()

	book := &Book{Title: doc.title(), Text: "", Pages: nil, Parts: nil, Structure: ""}

	var text strings.Builder
	pages := min(doc.pages, mostPages)
	starts := make([]int, 0, pages)
	for i := 0; i < pages; i++ {
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="text extraction halts when accumulated text reaches mostText"
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
		// Every page ends a line, so the last word of one page and the first word of
		// the next are separate words.
		if !strings.HasSuffix(page, "\n") {
			text.WriteString("\n")
		}
	}
	book.Text = text.String()

	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="document structure requires at least minimumParts entries"
	if named := doc.outline(starts); len(named) >= minimumParts {
		book.Parts, book.Structure = named, FromOutline
	} else {
		book.Structure = FromNothing
	}
	// gomutants:disable-next-line STATEMENT_REMOVE,RETURN_FALSE reason="parts are ordered by appearance in document text"
	sort.SliceStable(book.Parts, func(i, j int) bool {
		return book.Parts[i].Offset < book.Parts[j].Offset
	})
	return book
}

// A Location is where an offset in the text falls.
type Location struct {
	// Part is the nearest name at or before the offset, and empty when the
	// offset precedes every name.
	Part string
	// PartOffset is where that part begins.
	PartOffset int
	// the first. It is known for every page, and it is the one thing a page is
	// called.
	Page int
}

// Locate answers where one offset in the document's text is.
func (b *Book) Locate(offset int) Location {
	var loc Location
	if i := getPrecedingIndex(len(b.Parts), offset, func(i int) int { return b.Parts[i].Offset }); i >= 0 {
		loc.Part, loc.PartOffset = b.Parts[i].Title, b.Parts[i].Offset
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="page index matches when preceding index is non-negative"
	if i := getPrecedingIndex(len(b.Pages), offset, func(i int) int { return b.Pages[i].Offset }); i >= 0 {
		loc.Page = i
	}
	return loc
}

// getPrecedingIndex is the index of the last item at or before an offset, and -1 when
// there is none. The items are ascending by offset.
func getPrecedingIndex(n, offset int, offsetOf func(int) int) int {
	return sort.Search(n, func(i int) bool { return offsetOf(i) > offset }) - 1
}

// normalizeSpaces is one line of text: the words of a name, single-spaced.
func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
