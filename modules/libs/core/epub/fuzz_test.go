package epub_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/epub"
)

// mostPerBook is how much text one book may yield, and is the answer to an
// archive that says its entries are small and holds a few hundred kilobytes of
// zeros. It is the package's own bound, written again here because a target
// that reads the bound out of the code under test measures nothing.
const mostPerBook = 256 << 20

// mostFuzzed is how much of one book's text is read as markup before the target
// moves on to the next file.
const mostFuzzed = 1 << 20

// A book read without complaint is one the rest of the application can hold:
// every spine document, every named part and every printed page stands at an
// offset inside the text that was read, in the order they are read in, and each
// carries a name on one line. The tier the book says its parts came from is the
// tier that produced them.
//
// However small the file, the text it yields is bounded: what an archive says
// about the size of its entries is believed about nothing.
//
// A file that could not be read is refused as one that is no archive, one that
// holds no container, one that holds no package document, or one whose spine is
// ciphertext. A book that names nothing is read, and is a different answer.
func FuzzRead(f *testing.F) {
	whole := tinyBook(f, nil)
	f.Add(whole)
	// A file a copy stopped part way through, and one a byte of which was
	// written over.
	f.Add(whole[:len(whole)/2])
	f.Add(flipped(whole, len(whole)/3))
	// The central directory says one thing and the entry says another: the two
	// names disagree, which is what a shop's re-zipper leaves behind.
	f.Add(renamed(whole))
	// An entry whose header claims a size it has not got.
	f.Add(oversized(whole))

	// A book with no container, one with no package document, and one whose
	// package document is not a package document.
	f.Add(tinyBook(f, nil, "META-INF/container.xml"))
	f.Add(tinyBook(f, nil, "OEBPS/content.opf"))
	f.Add(tinyBook(f, map[string]string{"OEBPS/content.opf": "variant/garbage.opf"}))
	// A book whose places come from a navigation document rather than a control
	// file, and one whose only document carries no markup of its own.
	f.Add(tinyBook(f, map[string]string{
		"OEBPS/content.opf": "variant/nav.opf",
		"OEBPS/nav.xhtml":   "variant/nav.xhtml",
	}))
	f.Add(tinyBook(f, map[string]string{"OEBPS/first.xhtml": "variant/plain.xhtml"}))

	parts := tinyParts(f)
	// A spine naming a document that climbs out of the archive: once from a
	// package document in a folder, where what it names resolves to nothing and
	// the document is left out, and once from one at the root, where the archive
	// holds an entry under that very name and the book is read out of it.
	climbing := clone(parts)
	climbing["OEBPS/content.opf"] = bytes.ReplaceAll(
		climbing["OEBPS/content.opf"], []byte(`href="first.xhtml"`), []byte(`href="../../first.xhtml"`))
	climbing["../../first.xhtml"] = []byte(`<html><body><p>Out of the archive</p></body></html>`)
	f.Add(zipped(f, climbing))
	f.Add(zipped(f, map[string][]byte{
		"mimetype": []byte("application/epub+zip"),
		"META-INF/container.xml": []byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`),
		"content.opf": []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Out</dc:title></metadata>
  <manifest><item id="a" href="../../outside.xhtml" media-type="application/xhtml+xml"/></manifest>
  <spine><itemref idref="a"/></spine>
</package>`),
		"../../outside.xhtml": []byte(`<html><body><p>Out of the archive</p></body></html>`),
	}))

	// A book whose fonts are scrambled, and one whose first document is.
	for _, locked := range []string{"OEBPS/fonts/serif.otf", "OEBPS/first.xhtml"} {
		sealed := clone(parts)
		sealed["META-INF/encryption.xml"] = []byte(
			`<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
			  <EncryptedData xmlns="http://www.w3.org/2001/04/xmlenc#">
			    <CipherData><CipherReference URI="` + locked + `"/></CipherData>
			  </EncryptedData>
			</encryption>`)
		f.Add(zipped(f, sealed))
	}

	// A document saying it is written in one encoding and written in another.
	lying := clone(parts)
	lying["OEBPS/first.xhtml"] = []byte(
		"<?xml version=\"1.0\" encoding=\"UTF-16\"?>\n<html><body><p>Alpha</p></body></html>")
	f.Add(zipped(f, lying))

	// A document that is not well formed: a tag that never closes, and an
	// entity that is no entity.
	torn := clone(parts)
	torn["OEBPS/second.xhtml"] = []byte(`<html><body><p>Delta<div><span>&nope; &#xZZ;`)
	f.Add(zipped(f, torn))

	// An entry of nothing but zeros, which deflates to a few hundred bytes and
	// is read back as seventeen megabytes: a byte past what one document may
	// spend. The bound is what refuses it.
	bomb := clone(parts)
	bomb["OEBPS/second.xhtml"] = make([]byte, 17<<20)
	f.Add(zipped(f, bomb))

	// An archive that is no book, and nothing at all.
	f.Add(zipped(f, map[string][]byte{"notes.txt": []byte("a zip that is no book")}))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, raw []byte) {
		book, err := epub.Read(raw)
		if err != nil {
			named := errors.Is(err, epub.ErrNotArchive) ||
				errors.Is(err, epub.ErrNoContainer) ||
				errors.Is(err, epub.ErrNoPackage) ||
				errors.Is(err, epub.ErrEncrypted)
			if !named {
				t.Fatalf("a file of %d bytes was refused as %v, which says nothing about it",
					len(raw), err)
			}
			if book != nil {
				t.Fatalf("a file was refused as %v and a book came back with it", err)
			}
			return
		}
		if book == nil {
			t.Fatal("a file was read and no book came back")
		}
		if len(book.Text) > mostPerBook {
			t.Fatalf("%d bytes of archive were read as %d bytes of text",
				len(raw), len(book.Text))
		}

		at := -1
		for i, doc := range book.Documents {
			if doc.Length < 0 || doc.Offset < 0 || doc.Offset+doc.Length > len(book.Text) {
				t.Fatalf("document %d runs from %d for %d in %d bytes of text",
					i, doc.Offset, doc.Length, len(book.Text))
			}
			// A document holding nothing begins where the one before it ended,
			// so the order is what holds and not the distance.
			if doc.Offset < at {
				t.Fatalf("document %d begins at %d, before the one at %d", i, doc.Offset, at)
			}
			at = doc.Offset
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
			if part.Level < 0 {
				t.Fatalf("%q stands at depth %d", part.Title, part.Level)
			}
			if !oneLine(part.Title) {
				t.Fatalf("a part is named %q, which is not one line of a name", part.Title)
			}
		}

		at = -1
		for _, page := range book.Pages {
			if page.Offset < 0 || page.Offset > len(book.Text) {
				t.Fatalf("page %q begins at %d in %d bytes of text",
					page.Label, page.Offset, len(book.Text))
			}
			if page.Offset < at {
				t.Fatalf("page %q begins at %d, before the page at %d", page.Label, page.Offset, at)
			}
			at = page.Offset
			if !oneLine(page.Label) {
				t.Fatalf("a page is labelled %q, which is not one line of a label", page.Label)
			}
		}

		if told := book.Tier != epub.FromNothing; told != (len(book.Parts) >= 2) {
			t.Fatalf("%d parts came back and the book says they came from %q",
				len(book.Parts), book.Tier)
		}

		// Every offset a chunk of this text could carry is one the book can be
		// asked about, and what it answers with is what a reader is shown.
		for _, offset := range []int{0, len(book.Text) / 2, len(book.Text)} {
			where := book.Locate(offset)
			if where.Document != "" && !holds(book.Documents, where.Document, offset) {
				t.Fatalf("offset %d was put in %q, which is not a document standing over it",
					offset, where.Document)
			}
			if where.Part != "" {
				if where.PartOffset > offset {
					t.Fatalf("offset %d was put under %q, which begins at %d",
						offset, where.Part, where.PartOffset)
				}
				if !titled(book.Parts, where.Part, where.PartOffset) {
					t.Fatalf("offset %d was put under %q at %d, which the book does not name",
						offset, where.Part, where.PartOffset)
				}
			}
		}
	})
}

// oneLine is whether a name is the words of it, single-spaced, and not empty.
func oneLine(name string) bool {
	return name != "" && strings.Join(strings.Fields(name), " ") == name
}

// holds is whether the document of this name begins at or before the offset.
func holds(documents []epub.Document, at string, offset int) bool {
	for _, doc := range documents {
		if doc.Path == at && doc.Offset <= offset {
			return true
		}
	}
	return false
}

// titled is whether the book names this part, at this offset.
func titled(parts []epub.Part, title string, offset int) bool {
	for _, part := range parts {
		if part.Title == title && part.Offset == offset {
			return true
		}
	}
	return false
}

// clone is the files of a fixture, to be changed without changing the fixture.
func clone(parts map[string][]byte) map[string][]byte {
	made := make(map[string][]byte, len(parts))
	for name, raw := range parts {
		made[name] = raw
	}
	return made
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

// renamed is the archive with one entry named one thing in the central
// directory and another in the header the entry itself carries.
func renamed(raw []byte) []byte {
	changed := make([]byte, len(raw))
	copy(changed, raw)
	if at := bytes.Index(changed, []byte("OEBPS/first.xhtml")); at >= 0 {
		copy(changed[at:], []byte("OEBPS/other.xhtml"))
	}
	return changed
}

// oversized is the archive with one entry's central directory claiming far more
// bytes than the entry holds.
func oversized(raw []byte) []byte {
	changed := make([]byte, len(raw))
	copy(changed, raw)
	// The uncompressed size stands twenty-four bytes into a central directory
	// header, four bytes wide and least significant first.
	if at := bytes.Index(changed, []byte("PK\x01\x02")); at >= 0 && at+28 <= len(changed) {
		copy(changed[at+24:at+28], []byte{0xff, 0xff, 0xff, 0x7f})
	}
	return changed
}

// The markup of a document is the text of that document: the same walk writes
// both, and a reader takes its offsets off the markup and counts none itself. So
// whatever the file, the words of the markup are the words of the text, in
// order, and every element stands inside the document it was read from.
//
// A document the book was read from is always answered; nothing else is.
func FuzzMarkup(f *testing.F) {
	f.Add(tinyBook(f, nil))
	f.Add(tinyBook(f, map[string]string{"OEBPS/second.xhtml": "variant/drawn.xhtml"}))
	f.Add(tinyBook(f, map[string]string{"OEBPS/first.xhtml": "variant/plain.xhtml"}))

	// A document that is not well formed, and one whose markup is nothing but
	// elements the reader does not draw.
	torn := clone(tinyParts(f))
	torn["OEBPS/second.xhtml"] = []byte(`<html><body><p>Delta<div><span>&nope; &#xZZ;`)
	f.Add(zipped(f, torn))
	undrawn := clone(tinyParts(f))
	undrawn["OEBPS/first.xhtml"] = []byte(
		`<html><body><form><object><embed>Alpha</embed></object></form><iframe>Beta</iframe></body></html>`)
	f.Add(zipped(f, undrawn))

	f.Fuzz(func(t *testing.T, raw []byte) {
		book, err := epub.Read(raw)
		if err != nil {
			return
		}
		read := 0
		for _, doc := range book.Documents {
			// What is fuzzed is the reading of one document, so a book whose
			// every document is a megabyte of zeros is not read to the end.
			if read > mostFuzzed {
				break
			}
			read += doc.Length

			drawn, err := book.Markup(doc.Path)
			if err != nil {
				t.Fatalf("%s is a document of the book and was refused as %v", doc.Path, err)
			}
			if drawn.Offset != doc.Offset || drawn.Length != doc.Length {
				t.Fatalf("%s runs from %d for %d as markup and from %d for %d as text",
					doc.Path, drawn.Offset, drawn.Length, doc.Offset, doc.Length)
			}
			want := book.Text[doc.Offset : doc.Offset+doc.Length]
			if got := said(drawn.Nodes); got != want {
				t.Fatalf("the markup of %s says %d bytes and its text is %d",
					doc.Path, len(got), len(want))
			}
			shaped(t, doc, drawn.Nodes)
			checkSpans(t, book.Text, doc, drawn.Nodes, doc.Offset+doc.Length)
		}
		if _, err := book.Markup("nothing/the-spine-names.xhtml"); !errors.Is(err, epub.ErrNoDocument) {
			t.Fatalf("a document the book was not read from was answered with %v", err)
		}
	})
}

// shaped holds every node to being one thing: an element carries a name and
// holds what is under it, and a run carries text and holds nothing.
func shaped(t *testing.T, doc epub.Document, nodes []epub.Node) {
	t.Helper()
	for _, node := range nodes {
		if node.Name != "" && node.Text != "" {
			t.Fatalf("a %s element of %s carries text of its own", node.Name, doc.Path)
		}
		if node.Name == "" && len(node.Children) != 0 {
			t.Fatalf("a run of text in %s holds %d elements", doc.Path, len(node.Children))
		}
		shaped(t, doc, node.Children)
	}
}
