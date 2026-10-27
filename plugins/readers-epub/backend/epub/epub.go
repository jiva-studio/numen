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
// # One document is read a second time as the elements it is drawn from, carrying
//
// A book that is empty is still read. An error here means the file is not a
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

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/archive"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/container"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/navigation"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/opf"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/text"
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

	Layout Layout

	// Direction is the direction the book's pages progress in, and empty for a
	// book that says nothing.
	Direction Direction

	// files is the archive the book was read out of, which Markup reads one
	// document out of again, and documents is the document of each name.
	files      map[string]*zip.File
	documents  map[string]Document
	mediaTypes map[string]string

	// spineStep and itemSteps are the CFI steps of the spine in the package document and of each
	// itemref in the spine, by spine index.
	spineStep int
	itemSteps []int

	// pageBytes is how many bytes of this book's text stand on one page.
	pageBytes int
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

// The four ways a file can fail to be a book, and the one way a document of one
// can fail to be asked for.
var (
	ErrNotArchive  = errors.New("epub: not a zip archive")
	ErrNoContainer = container.ErrNoContainer
	ErrNoPackage   = opf.ErrNoPackage
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

	extractor := text.NewExtractor()
	book := &Book{
		Title:      pkg.Title,
		Layout:     pkg.Layout,
		Direction:  pkg.Direction,
		files:      files,
		documents:  map[string]Document{},
		mediaTypes: map[string]string{},
		spineStep:  pkg.SpineStep,
	}
	remaining := int64(archive.MostPerBook)
	for _, item := range pkg.Spine {
		book.itemSteps = append(book.itemSteps, item.Step)
	}
	for spine, item := range pkg.Spine {
		if remaining <= 0 {
			// gomutants:disable-next-line INVERT_LOOP_CTRL reason="left never grows, so every later document also ends the iteration"
			break
		}
		raw, ok := archive.ReadBounded(files[item.Path], min(int64(archive.MostPerDocument), remaining))
		if !ok {
			// A manifest may name a file the archive does not hold, and one it
			// holds may be larger than a chapter can be.
			continue
		}
		remaining -= int64(len(raw))
		doc := extractor.ReadDocument(item.Path, raw)
		doc.IsLinear, doc.Layout, doc.Spine = item.IsLinear, item.Layout, spine
		book.Documents = append(book.Documents, doc)
		if _, twice := book.documents[doc.Path]; !twice {
			// A spine may read one document twice, and the first standing is the
			// one a name is answered with.
			book.documents[doc.Path] = doc
			book.mediaTypes[doc.Path] = item.MediaType
		}
	}
	book.Text = string(extractor.GetText())
	book.pageBytes = computePageBytes(book.Text)

	named := navigation.ReadParts(files, pkg, extractor)
	switch {
	case len(named) >= navigation.MinimumParts:
		book.Parts, book.Tier = named, FromNavigation
	case len(extractor.GetHeadings()) >= navigation.MinimumParts:
		book.Parts, book.Tier = extractor.GetHeadings(), FromHeadings
	default:
		book.Tier = FromNothing
	}
	sort.SliceStable(book.Parts, func(i, j int) bool {
		return book.Parts[i].Offset < book.Parts[j].Offset
	})

	book.Pages = extractor.GetPages(navigation.ReadPageEntries(files, pkg))
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
func openPackage(r io.ReaderAt, size int64) (map[string]*zip.File, opf.Package, error) {
	reader, err := zip.NewReader(r, size)
	if err != nil {
		return nil, opf.Package{}, fmt.Errorf("%w: %w", ErrNotArchive, err)
	}
	files := archive.CreateIndex(reader)

	opfPath, err := container.ReadPackagePath(files)
	if err != nil {
		return nil, opf.Package{}, err
	}
	pkg, err := opf.Read(files, opfPath)
	if err != nil {
		return nil, opf.Package{}, err
	}
	locked := container.ReadEncryptedPaths(files)
	for _, item := range pkg.Spine {
		if locked[item.Path] {
			return nil, opf.Package{}, fmt.Errorf("%w: %s", ErrEncrypted, item.Path)
		}
	}
	return files, pkg, nil
}

func readEntry(files map[string]*zip.File, name string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: the archive is not named", ErrNoEntry)
	}
	raw, ok := archive.ReadBounded(files[path.Clean(name)], archive.MostPerDocument)
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
	var loc Location
	if i := findPreceding(len(b.Documents), offset, func(i int) int { return b.Documents[i].Offset }); i >= 0 {
		loc.Document = b.Documents[i].Path
	}
	if i := findPreceding(len(b.Parts), offset, func(i int) int { return b.Parts[i].Offset }); i >= 0 {
		loc.Part, loc.PartOffset = b.Parts[i].Title, b.Parts[i].Offset
	}
	if i := findPreceding(len(b.Pages), offset, func(i int) int { return b.Pages[i].Offset }); i >= 0 {
		loc.Page = b.Pages[i].Label
	}
	return loc
}

// findPreceding is the index of the last item at or before an offset, and -1 when
// there is none. The items are ascending by offset.
func findPreceding(n, offset int, offsetOf func(int) int) int {
	return sort.Search(n, func(i int) bool { return offsetOf(i) > offset }) - 1
}
