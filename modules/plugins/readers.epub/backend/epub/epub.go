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
// One document is read a second time as the elements it is drawn from, carrying
// the offsets its text already stands at.
//
// A book that names nothing is still read. An error here means the file is not a
// book at all.
package epub

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
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

	// Layout is whether the book's documents can be reflowed.
	Layout Layout

	// Direction is the direction the book's pages progress in, and empty for a
	// book that says nothing.
	Direction Direction

	// files is the archive the book was read out of, which Markup reads one
	// document out of again, and held is the document of each name.
	files map[string]*zip.File
	held  map[string]Document

	// pageBytes is how many bytes of this book's text stand on one page.
	pageBytes int
}

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
	// Layout is whether this document can be reflowed.
	Layout Layout

	// mediaType is what the manifest calls the document.
	mediaType string
}

// A Layout is whether a document can be reflowed or is a page laid out once and
// drawn as it stands.
type Layout string

const (
	Reflowable   Layout = "reflowable"
	PrePaginated Layout = "pre-paginated"
)

// A Direction is the direction a book's pages progress in.
type Direction string

const (
	// DefaultDirection: the book says nothing, and a reader lays it out the way
	// its language is written.
	DefaultDirection Direction = ""
	LeftToRight      Direction = "ltr"
	RightToLeft      Direction = "rtl"
)

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

// The four ways a file can fail to be a book, and the one way a document of one
// can fail to be asked for.
var (
	ErrNotArchive  = errors.New("epub: not a zip archive")
	ErrNoContainer = errors.New("epub: no META-INF/container.xml")
	ErrNoPackage   = errors.New("epub: no package document")
	// ErrEncrypted: a document of the spine is ciphertext, and what would be
	// read out of it is not the book.
	ErrEncrypted = errors.New("epub: a spine document is encrypted")
	// ErrNoDocument: the book was not read from a document of this name.
	ErrNoDocument = errors.New("epub: no such spine document")
	// ErrNoEntry: the archive holds nothing readable under this name.
	ErrNoEntry = errors.New("epub: no such entry")
)

// Read extracts one book from the bytes of an EPUB file.
func Read(raw []byte) (*Book, error) {
	files, pkg, err := openPackage(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}

	text := newExtractor()
	book := &Book{
		Title:     pkg.title,
		Layout:    pkg.layout,
		Direction: pkg.direction,
		files:     files,
		held:      map[string]Document{},
	}
	left := int64(mostPerBook)
	for _, item := range pkg.spine {
		if left <= 0 {
			// gomutants:disable-next-line INVERT_LOOP_CTRL reason="left never grows, so every later document also ends the iteration"
			break
		}
		markup, ok := readBounded(files[item.path], min(int64(mostPerDocument), left))
		if !ok {
			// A manifest may name a file the archive does not hold, and one it
			// holds may be larger than a chapter can be.
			continue
		}
		left -= int64(len(markup))
		doc := text.readDocument(item.path, markup)
		doc.IsLinear, doc.Layout, doc.mediaType = item.isLinear, item.layout, item.mediaType
		book.Documents = append(book.Documents, doc)
		if _, twice := book.held[doc.Path]; !twice {
			// A spine may read one document twice, and the first standing is the
			// one a name is answered with.
			book.held[doc.Path] = doc
		}
	}
	book.Text = string(text.out)
	book.pageBytes = computePageBytes(book.Text)

	named := readNavigationParts(files, pkg, text)
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

	book.Pages = text.getPages(readPageEntries(files, pkg))
	sort.SliceStable(book.Pages, func(i, j int) bool {
		return book.Pages[i].Offset < book.Pages[j].Offset
	})
	return book, nil
}

// Entry is the bytes of one entry of the archive, which is how a picture a book
// carries is drawn. An entry larger than a document is read at is not there.
//
// What the entry holds is the file's own bytes, and what the manifest calls it
// is not among them.
func (b *Book) Entry(name string) ([]byte, error) {
	return readEntry(b.files, name)
}

// openPackage indexes the archive behind r and checks that it contains a book whose
// spine is readable.
func openPackage(r io.ReaderAt, size int64) (map[string]*zip.File, packageDoc, error) {
	archive, err := zip.NewReader(r, size)
	if err != nil {
		return nil, packageDoc{}, fmt.Errorf("%w: %w", ErrNotArchive, err)
	}
	files := createArchiveIndex(archive)

	opfPath, err := readPackagePath(files)
	if err != nil {
		return nil, packageDoc{}, err
	}
	pkg, err := readPackage(files, opfPath)
	if err != nil {
		return nil, packageDoc{}, err
	}
	locked := readEncryptedPaths(files)
	for _, item := range pkg.spine {
		if locked[item.path] {
			return nil, packageDoc{}, fmt.Errorf("%w: %s", ErrEncrypted, item.path)
		}
	}
	return files, pkg, nil
}

func readEntry(files map[string]*zip.File, name string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: the archive is not named", ErrNoEntry)
	}
	raw, ok := readBounded(files[path.Clean(name)], mostPerDocument)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoEntry, name)
	}
	return raw, nil
}

// An Archive is the index of the entries of one EPUB file, read through a reader that
// stays open. Only the entry asked for is read, and any number of goroutines may
// ask at once while the reader serves concurrent reads.
type Archive struct {
	files map[string]*zip.File
}

// OpenArchive indexes the EPUB behind r, of size bytes, and refuses what
// Read refuses.
func OpenArchive(r io.ReaderAt, size int64) (*Archive, error) {
	files, _, err := openPackage(r, size)
	if err != nil {
		return nil, err
	}
	return &Archive{files: files}, nil
}

// Entry is the bytes of one entry of the archive, as Book.Entry answers them.
func (a *Archive) Entry(name string) ([]byte, error) {
	return readEntry(a.files, name)
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
	if i := findPreceding(len(b.Documents), offset, func(i int) int { return b.Documents[i].Offset }); i >= 0 {
		at.Document = b.Documents[i].Path
	}
	if i := findPreceding(len(b.Parts), offset, func(i int) int { return b.Parts[i].Offset }); i >= 0 {
		at.Part, at.PartOffset = b.Parts[i].Title, b.Parts[i].Offset
	}
	if i := findPreceding(len(b.Pages), offset, func(i int) int { return b.Pages[i].Offset }); i >= 0 {
		at.Page = b.Pages[i].Label
	}
	return at
}

// findPreceding is the index of the last item at or before an offset, and -1 when
// there is none. The items are ascending by offset.
func findPreceding(n, offset int, offsetOf func(int) int) int {
	return sort.Search(n, func(i int) bool { return offsetOf(i) > offset }) - 1
}

// normaliseSpaces is one line of text: the words of a name, single-spaced.
func normaliseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
