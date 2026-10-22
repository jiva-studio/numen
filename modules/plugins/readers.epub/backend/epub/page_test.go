package epub_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub"
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
		pos := book.PageStart(page)
		if pos < 0 || pos > len(book.Text) {
			t.Fatalf("page %d begins at %d in %d bytes of text", page, pos, len(book.Text))
		}
		if got := book.PageOf(pos); got != page {
			t.Errorf("page %d begins at %d, which is on page %d", page, pos, got)
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
	content := func(letter string) *epub.Book {
		t.Helper()
		return read(t, buildSpineArchive(t, oneDocumentOpf, map[string]string{
			"OEBPS/one.xhtml": "<html><body><p>" + strings.Repeat(letter, 20_000) + "</p></body></html>",
		}))
	}

	latin, cyrillic := content("a"), content("б")
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
		pos := cyrillic.PageStart(page)
		if r, _ := utf8.DecodeRuneInString(cyrillic.Text[pos:]); r == utf8.RuneError {
			t.Fatalf("page %d begins at %d, in the middle of a letter", page, pos)
		}
	}
}

const oneDocumentOpf = `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>One</dc:title></metadata>
  <manifest><item id="a" href="one.xhtml" media-type="application/xhtml+xml"/></manifest>
  <spine><itemref idref="a"/></spine>
</package>`

func textOfBytes(n int) *epub.Book {
	return &epub.Book{Text: strings.Repeat("a", n)}
}

// A book whose text has not been measured is read a page of 1024 letters at a time.
func TestAPageOfUnmeasuredTextIsAKibibyte(t *testing.T) {
	if got := textOfBytes(0).PageBytes(); got != 1024 {
		t.Errorf("a page is %d bytes, want 1024", got)
	}

	counts := map[int]int{0: 1, 1: 1, 1023: 1, 1024: 1, 1025: 2, 2048: 2, 2049: 3}
	for size, want := range counts {
		if got := textOfBytes(size).PageCount(); got != want {
			t.Errorf("%d bytes are %d pages, want %d", size, got, want)
		}
	}

	book := textOfBytes(3000)
	offsets := map[int]int{-2000: 1, -1: 1, 0: 1, 1023: 1, 1024: 2, 2047: 2, 2048: 3, 2999: 3, 9000: 3}
	for offset, want := range offsets {
		if got := book.PageOf(offset); got != want {
			t.Errorf("offset %d is on page %d, want %d", offset, got, want)
		}
	}

	starts := map[int]int{-3: 0, 0: 0, 1: 0, 2: 1024, 3: 2048, 4: 3000, 100: 3000}
	for page, want := range starts {
		if got := book.PageStart(page); got != want {
			t.Errorf("page %d begins at %d, want %d", page, got, want)
		}
	}
}

// A book with no letters at all is still one page of 1024 bytes.
func TestAPageOfABookWithoutTextIsAKibibyte(t *testing.T) {
	book := read(t, buildSpineArchive(t, oneDocumentOpf, map[string]string{
		"OEBPS/one.xhtml": "<html><body></body></html>",
	}))
	if got := book.PageBytes(); got != 1024 {
		t.Errorf("a page is %d bytes, want 1024", got)
	}
	if got := book.PageCount(); got != 1 {
		t.Errorf("a book without text is %d pages, want 1", got)
	}
}

// A page is measured in the script of its own book, and one that opens inside a
// letter moves on to the next.
func TestAPageBeginsAtTheFirstLetterAtOrAfterItsMeasure(t *testing.T) {
	scripts := map[string]struct {
		letters string
		bytes   int
	}{
		"latin":    {"a", 1024},
		"cyrillic": {"б", 2047},
		"mixed":    {"aa" + strings.Repeat("б", 20_000), 2047},
	}
	for name, script := range scripts {
		t.Run(name, func(t *testing.T) {
			body := script.letters
			if name != "mixed" {
				body = strings.Repeat(script.letters, 20_000)
			}
			book := read(t, buildSpineArchive(t, oneDocumentOpf, map[string]string{
				"OEBPS/one.xhtml": "<html><body><p>" + body + "</p></body></html>",
			}))
			if got := book.PageBytes(); got != script.bytes {
				t.Fatalf("a page is %d bytes, want %d", got, script.bytes)
			}
			for page := 1; page <= book.PageCount(); page++ {
				measure := (page - 1) * book.PageBytes()
				pos := book.PageStart(page)
				if pos < measure || pos-measure >= utf8.UTFMax {
					t.Fatalf("page %d begins at %d, measure %d", page, pos, measure)
				}
				if !utf8.RuneStart(book.Text[pos]) {
					t.Fatalf("page %d begins inside a letter at %d", page, pos)
				}
				if pos > 0 && utf8.RuneStart(book.Text[measure]) && pos != measure {
					t.Fatalf("page %d moved off a letter, %d to %d", page, measure, pos)
				}
			}
		})
	}
}
