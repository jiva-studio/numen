package pdf_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// FuzzRead checks that reading arbitrary bytes never fails the invariants the
// rest of the application relies on. A document that reads successfully lists
// its parts in pagination order, and each part has a one-line name; a document
// whose outline has no names reports that.
//
// Locating an offset yields the page that begins at or before it and the part
// that begins at or before it, so a passage found in the text belongs to a page
// and a heading.
//
// A file that cannot be read fails as not a PDF or as password-protected. A scan
// has no text and is a successfully read document.
//
// The input bytes are a file in a user's vault, written by a scanner, a browser
// or a sync tool, and possibly incomplete.
func FuzzRead(f *testing.F) {
	for _, name := range []string{tiny, outline, labels, turned, scan, vast, prose} {
		raw, err := os.ReadFile("testdata/" + name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(raw)
		// A file a copy stopped part way through, and one a byte of which was
		// written over. Both are what a vault holds after a sync was
		// interrupted, and neither announces itself.
		f.Add(raw[:len(raw)/2])
		f.Add(flipByte(raw, len(raw)/3))
	}
	// A header and nothing behind it, a header naming a version there is not,
	// a body with no header, and nothing at all.
	f.Add([]byte("%PDF-1.7\n"))
	f.Add([]byte("%PDF-9.9\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%EOF\n"))
	f.Add([]byte("1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF\n"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, raw []byte) {
		book, err := pdf.Read(raw)
		if err != nil {
			if !errors.Is(err, pdf.ErrNotPDF) && !errors.Is(err, pdf.ErrEncrypted) {
				t.Fatalf("a file of %d bytes failed as %v, which says nothing about it",
					len(raw), err)
			}
			if book != nil {
				t.Fatalf("a file failed as %v and a document came back with it", err)
			}
			return
		}
		if book == nil {
			t.Fatal("a file was read and no document came back")
		}

		pos := -1
		for i, page := range book.Pages {
			if page.Offset < 0 || page.Offset > len(book.Text) {
				t.Fatalf("page %d begins at %d in %d bytes of text",
					i, page.Offset, len(book.Text))
			}
			// Two pages that say nothing begin in the same place, so the order
			// is what holds and not the distance.
			if page.Offset < pos {
				t.Fatalf("page %d begins at %d, before page %d at %d",
					i, page.Offset, i-1, pos)
			}
			pos = page.Offset
		}

		pos = -1
		for _, part := range book.Parts {
			if part.Offset < 0 || part.Offset > len(book.Text) {
				t.Fatalf("%q begins at %d in %d bytes of text",
					part.Title, part.Offset, len(book.Text))
			}
			if part.Offset < pos {
				t.Fatalf("%q begins at %d, before the part at %d", part.Title, part.Offset, pos)
			}
			pos = part.Offset
			if part.Level < 1 {
				t.Fatalf("part %q is at depth %d", part.Title, part.Level)
			}
			if part.Title == "" || strings.Join(strings.Fields(part.Title), " ") != part.Title {
				t.Fatalf("a part is named %q, which is not one line of a name", part.Title)
			}
		}

		if named := book.Structure == pdf.FromOutline; named != (len(book.Parts) >= 2) {
			t.Fatalf("%d parts came back and the document says they came from %q",
				len(book.Parts), book.Structure)
		}

		// Every offset a chunk of this text can carry is a valid query for the
		// document, and the page and part returned are the ones shown to the reader.
		for _, offset := range []int{0, len(book.Text) / 2, len(book.Text)} {
			where := book.Locate(offset)
			if len(book.Pages) > 0 {
				if where.Page < 0 || where.Page >= len(book.Pages) {
					t.Fatalf("offset %d falls on page %d of %d",
						offset, where.Page, len(book.Pages))
				}
				if book.Pages[where.Page].Offset > offset {
					t.Fatalf("offset %d was put on the page beginning at %d",
						offset, book.Pages[where.Page].Offset)
				}
			}
			if where.Part != "" {
				if where.PartOffset > offset {
					t.Fatalf("offset %d was put under %q, which begins at %d",
						offset, where.Part, where.PartOffset)
				}
				if !hasPart(book.Parts, where.Part, where.PartOffset) {
					t.Fatalf("offset %d was put under %q at %d, which the outline does not name",
						offset, where.Part, where.PartOffset)
				}
			}
		}
	})
}

// hasPart is whether the outline holds this part, at this offset.
func hasPart(parts []pdf.Part, title string, offset int) bool {
	for _, part := range parts {
		if part.Title == title && part.Offset == offset {
			return true
		}
	}
	return false
}

// flipByte is the file with one byte of it written over by another.
func flipByte(raw []byte, pos int) []byte {
	if len(raw) == 0 {
		return raw
	}
	changed := make([]byte, len(raw))
	copy(changed, raw)
	changed[pos%len(changed)] ^= 0xff
	return changed
}
