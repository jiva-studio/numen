package epub_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/epub"
)

// A reflowable book has no pages of its own, and laying one out to count them is
// what makes a reader slow to open a book. The pages are counted over the text,
// which is already one stream of bytes.
func TestThePagesABookIsRead(t *testing.T) {
	book := read(t, tinyBook(t, nil))

	if book.PageCount() < 1 {
		t.Fatalf("pages = %d, want the page a person is looking at", book.PageCount())
	}
	if got := book.PageOf(0); got != 1 {
		t.Errorf("the first byte is on page %d", got)
	}
	if got := book.PageOf(len(book.Text) - 1); got != book.PageCount() {
		t.Errorf("the last byte is on page %d of %d", got, book.PageCount())
	}
	for page := 1; page <= book.PageCount(); page++ {
		at := book.PageStart(page)
		if at < 0 || at > len(book.Text) {
			t.Fatalf("page %d begins at %d in %d bytes of text", page, at, len(book.Text))
		}
		if got := book.PageOf(at); got != page {
			t.Errorf("page %d begins at %d, which is on page %d", page, at, got)
		}
	}
	// An offset outside the text is still answered, because a caller holding a
	// stale offset is shown a page and not a panic.
	if got := book.PageOf(len(book.Text) * 2); got != book.PageCount() {
		t.Errorf("an offset past the text is on page %d of %d", got, book.PageCount())
	}
}

// A page is a number of letters, not of bytes, so the same book in two scripts
// is the same number of pages, and a page of either begins at a letter.
func TestAPageIsThatManyLetters(t *testing.T) {
	written := func(letter string) *epub.Book {
		t.Helper()
		return read(t, spined(t, oneDocumentOpf, map[string]string{
			"OEBPS/one.xhtml": "<html><body><p>" + strings.Repeat(letter, 20_000) + "</p></body></html>",
		}))
	}

	latin, cyrillic := written("a"), written("б")
	if latin.PageCount() != cyrillic.PageCount() {
		t.Errorf("twenty thousand letters are %d pages in Latin and %d in Cyrillic",
			latin.PageCount(), cyrillic.PageCount())
	}
	if got := latin.PageCount(); got < 15 || got > 25 {
		t.Errorf("twenty thousand letters are %d pages, want about twenty", got)
	}
	// A page opened in the middle of a letter is a page of broken text, and an
	// offset that names no place in the book.
	for page := 1; page <= cyrillic.PageCount(); page++ {
		at := cyrillic.PageStart(page)
		if r, _ := utf8.DecodeRuneInString(cyrillic.Text[at:]); r == utf8.RuneError {
			t.Fatalf("page %d begins at %d, in the middle of a letter", page, at)
		}
	}
}

const oneDocumentOpf = `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>One</dc:title></metadata>
  <manifest><item id="a" href="one.xhtml" media-type="application/xhtml+xml"/></manifest>
  <spine><itemref idref="a"/></spine>
</package>`
