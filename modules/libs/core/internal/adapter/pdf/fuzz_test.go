package pdf_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
)

// A document read without complaint is one the rest of the application can
// hold: every page and every named part stands at an offset in the text that
// was read, in the order they are paginated, and each part carries a name on
// one line. Where the outline named nothing, the document says so rather than
// standing an empty list beside a structure that claims one.
//
// Locating an offset lands on a page that begins at or before it and on the
// part that begins at or before it, so a passage found in the text can be said
// to be on a page and under a heading — which is the whole of what a reader is
// shown about where it came from.
//
// A file that could not be read is refused as one that is no PDF or as one
// nobody here has the password to, and never as a document with nothing in it:
// a scan carries no text either, and the two are not the same answer.
//
// The bytes are a file in a person's vault, put there by a scanner, a browser
// or a sync, and half-written by any of them.
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
		f.Add(flipped(raw, len(raw)/3))
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
				t.Fatalf("a file of %d bytes was refused as %v, which says nothing about it",
					len(raw), err)
			}
			if book != nil {
				t.Fatalf("a file was refused as %v and a document came back with it", err)
			}
			return
		}
		if book == nil {
			t.Fatal("a file was read and no document came back")
		}

		at := -1
		for i, page := range book.Pages {
			if page.Offset < 0 || page.Offset > len(book.Text) {
				t.Fatalf("page %d begins at %d in %d bytes of text",
					i, page.Offset, len(book.Text))
			}
			// Two pages that say nothing begin in the same place, so the order
			// is what holds and not the distance.
			if page.Offset < at {
				t.Fatalf("page %d begins at %d, before page %d at %d",
					i, page.Offset, i-1, at)
			}
			at = page.Offset
		}

		at = -1
		for _, part := range book.Parts {
			if part.Offset < 0 || part.Offset > len(book.Text) {
				t.Fatalf("%q begins at %d in %d bytes of text",
					part.Title, part.Offset, len(book.Text))
			}
			if part.Offset < at {
				t.Fatalf("%q begins at %d, before the part at %d", part.Title, part.Offset, at)
			}
			at = part.Offset
			if part.Level < 1 {
				t.Fatalf("%q stands at depth %d", part.Title, part.Level)
			}
			if part.Title == "" || strings.Join(strings.Fields(part.Title), " ") != part.Title {
				t.Fatalf("a part is named %q, which is not one line of a name", part.Title)
			}
		}

		if named := book.Structure == pdf.FromOutline; named != (len(book.Parts) >= 2) {
			t.Fatalf("%d parts came back and the document says they came from %q",
				len(book.Parts), book.Structure)
		}

		// Every offset a chunk of this text could carry is one the document can
		// be asked about, and the page and the part it answers with are the
		// ones a reader is shown.
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
				if !named(book.Parts, where.Part, where.PartOffset) {
					t.Fatalf("offset %d was put under %q at %d, which the outline does not name",
						offset, where.Part, where.PartOffset)
				}
			}
		}
	})
}

// named is whether the outline holds this part, at this offset.
func named(parts []pdf.Part, title string, offset int) bool {
	for _, part := range parts {
		if part.Title == title && part.Offset == offset {
			return true
		}
	}
	return false
}

// flipped is the file with one byte of it written over by another.
func flipped(raw []byte, at int) []byte {
	if len(raw) == 0 {
		return raw
	}
	changed := make([]byte, len(raw))
	copy(changed, raw)
	changed[at%len(changed)] ^= 0xff
	return changed
}
