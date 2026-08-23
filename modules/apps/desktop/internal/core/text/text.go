// Package text answers one question for every kind of source: what does this
// file say, and where in it is any offset.
//
// Three callers ask it — an extractor cutting a source, a search showing a
// passage, an embedder re-slicing a window — and they have to agree. A chunk
// keeps an offset into the text a reader produced; a second reader producing
// other text at other offsets reads the wrong place and says so with
// confidence.
//
// Reading one file's bytes is pure: no filesystem, no clock, no database, and
// the same bytes give the same text at the same offsets. Reader is the part
// that fetches those bytes, and it is the only part that touches anything.
package text

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/epub"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/pdf"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/window"
)

// ErrUnreadable is a file that is there and says nothing this can use. It is
// not a failure of whatever asked: one file is one file, and a library goes on.
var ErrUnreadable = errors.New("nothing could be read from the source")

// A Document is a source, read.
type Document struct {
	// Text is what the source says, as one stream. Every offset below is a byte
	// offset into it.
	Text string

	// Places are where the source names something, and are what windows are cut
	// inside so that one never runs across a part into the next.
	Places []window.Place

	// named and paged are what the source calls the place an offset falls in,
	// in the vocabulary of its own format. Both ascend by offset, and either may
	// be empty: half the books read name neither.
	named []mark
	paged []mark
}

// A mark is somewhere the source gives a name to.
type mark struct {
	Offset int
	Name   string
}

// Locate is where an offset is: the part the source's own navigation, outline
// or headings name, and the page it falls on. Empty where the source has
// neither.
func (d *Document) Locate(offset int) string {
	named := make([]string, 0, 2)
	if i := preceding(d.named, offset); i >= 0 {
		named = append(named, d.named[i].Name)
	}
	if i := preceding(d.paged, offset); i >= 0 {
		named = append(named, d.paged[i].Name)
	}
	return strings.Join(named, ", ")
}

// Opens are the parts that begin exactly at an offset: what a section starting
// here is called.
//
// A part and the first subsection inside it can begin at one place, and both
// name it. What is answered is every name, outermost first, so a question about
// either reaches the same place.
func (d *Document) Opens(offset int) []string {
	var names []string
	for _, m := range d.named {
		if m.Offset == offset {
			names = append(names, m.Name)
		}
		if m.Offset > offset {
			break
		}
	}
	return names
}

// sheet is what a page of a file is called: where it stands in it.
//
// A person is told the number a viewer opens at, so there is one number and it
// is the one on the screen. What the paper printed is a second number for the
// same page, and a person shown both has to work out which is being talked
// about.
func sheet(at int) string {
	return fmt.Sprintf("page %d of the file", at+1)
}

func preceding(marks []mark, offset int) int {
	return sort.Search(len(marks), func(i int) bool { return marks[i].Offset > offset }) - 1
}

// Readers are the names of what takes text out of a file. A name is part of a
// source's recipe and changes when the text or the offsets it produces do.
const (
	ReaderNote = "note-1"
	ReaderEPUB = "epub-1"
	ReaderPDF  = "pdf-1"
)

// ReaderName names what would read this file. A file nothing reads has no name,
// and nothing asks for its text.
func ReaderName(ref domain.FileRef) (string, bool) {
	if ref.Kind == domain.KindNote {
		return ReaderNote, true
	}
	switch strings.ToLower(path.Ext(ref.Path)) {
	case ".epub":
		return ReaderEPUB, true
	case ".pdf":
		return ReaderPDF, true
	}
	return "", false
}

// Read is what one file says.
//
// A note is its own bytes. A book is what taking the text out of it produces,
// and a chunk's offsets belong to that and not to the bytes on disk: slicing an
// archive at a text offset returns compressed noise.
func Read(ref domain.FileRef, raw []byte) (*Document, error) {
	reader, ok := ReaderName(ref)
	if !ok {
		return nil, ErrUnreadable
	}
	switch reader {
	case ReaderNote:
		return &Document{Text: string(raw)}, nil
	case ReaderEPUB:
		return fromEPUB(raw)
	case ReaderPDF:
		return fromPDF(raw)
	}
	return nil, ErrUnreadable
}

func fromEPUB(raw []byte) (*Document, error) {
	book, err := epub.Read(raw)
	if err != nil {
		return nil, ErrUnreadable
	}
	doc := &Document{Text: book.Text}
	for _, p := range book.Places {
		doc.Places = append(doc.Places, window.Place{Title: p.Title, Offset: p.Offset})
		doc.named = append(doc.named, mark{Offset: p.Offset, Name: p.Title})
	}
	// A book made for a screen has no pages of its own, and those it names are
	// the printed edition it was set from. That is the only name they have.
	for _, p := range book.Pages {
		if p.Label == "" {
			continue
		}
		doc.paged = append(doc.paged, mark{Offset: p.Offset, Name: p.Label})
	}
	return doc, nil
}

func fromPDF(raw []byte) (*Document, error) {
	book, err := pdf.Read(raw)
	if err != nil {
		return nil, ErrUnreadable
	}
	doc := &Document{Text: book.Text}
	for _, p := range book.Places {
		doc.Places = append(doc.Places, window.Place{Title: p.Title, Offset: p.Offset})
		doc.named = append(doc.named, mark{Offset: p.Offset, Name: p.Title})
	}
	for i, p := range book.Pages {
		doc.paged = append(doc.paged, mark{Offset: p.Offset, Name: sheet(i)})
	}
	return doc, nil
}
