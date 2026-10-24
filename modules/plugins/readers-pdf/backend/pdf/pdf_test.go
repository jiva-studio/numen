package pdf_test

import (
	"bytes"
	"errors"
	"image"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/plugins/readers-pdf/backend/pdf"
)

// The fixtures are six documents, and each is one thing a reader has to get
// right.
//
// Five of them are written out by hand, uncompressed and without a cross
// reference table, which is what a great many files in the world look like
// after a tool has half-written them: the library rebuilds what it needs, and a
// person can read the fixture. `outline.pdf` is the exception. An outline's
// destinations resolve only in a file whose cross reference table is real, so
// that one is produced and not typed, and it is still plain text.
const (
	tiny    = "tiny.pdf"    // two pages of text, naming nothing
	outline = "outline.pdf" // four pages, three of them named, one name inside another
	labels  = "labels.pdf"  // front matter numbered apart from the body
	turned  = "turned.pdf"  // a page drawn a quarter turn from the way its text is written
	scan    = "scan.pdf"    // two pages that carry no text at all
	vast    = "vast.pdf"    // one page of the largest box the format allows
	prose   = "prose.txt"   // not a document
)

func read(t *testing.T, name string) *pdf.Book {
	t.Helper()
	book, err := pdf.Read(fixture(t, name))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return book
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestPagesAreTheReadingOrder(t *testing.T) {
	book := read(t, tiny)

	want := "Alpha beta gamma\nDelta epsilon zeta\n"
	if book.Text != want {
		t.Errorf("text is %q, want %q", book.Text, want)
	}
	if len(book.Pages) != 2 {
		t.Fatalf("%d pages, want 2", len(book.Pages))
	}
	if book.Pages[0].Offset != 0 {
		t.Errorf("the first page begins at %d, want 0", book.Pages[0].Offset)
	}
	if pos := book.Pages[1].Offset; !strings.HasPrefix(book.Text[pos:], "Delta") {
		t.Errorf("the second page begins at %d, which is %q", pos, book.Text[pos:])
	}
}

func TestAPageIsWhereItExistsInTheFile(t *testing.T) {
	// A page carries no name of its own. A document numbering its front matter
	// apart from its body still has three pages, and they stand in one order.
	book := read(t, labels)

	if len(book.Pages) != 3 {
		t.Fatalf("%d pages, want 3", len(book.Pages))
	}
	for i, page := range book.Pages {
		if i > 0 && page.Offset < book.Pages[i-1].Offset {
			t.Errorf("page %d begins before the page before it", i)
		}
	}
}

func TestStructureReachesFromTheOutlineOrFromNothing(t *testing.T) {
	tests := []struct {
		name  string
		file  string
		want  pdf.Structure
		parts []string
	}{
		{
			name:  "an outline names the parts",
			file:  outline,
			want:  pdf.FromOutline,
			parts: []string{"The First Part", "A Closer Reading", "Afterword"},
		},
		{
			name: "a document with no outline names nothing",
			file: tiny,
			want: pdf.FromNothing,
		},
		{
			name: "a scan names nothing either",
			file: scan,
			want: pdf.FromNothing,
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			book := read(t, c.file)
			if book.Structure != c.want {
				t.Errorf("structure is %q, want %q", book.Structure, c.want)
			}
			var named []string
			for _, p := range book.Parts {
				named = append(named, p.Title)
			}
			if len(named) != len(c.parts) {
				t.Fatalf("parts are %q, want %q", named, c.parts)
			}
			for i := range named {
				if named[i] != c.parts[i] {
					t.Errorf("part %d is %q, want %q", i, named[i], c.parts[i])
				}
			}
		})
	}
}

func TestAPartCarriesTheDepthTheOutlineGaveIt(t *testing.T) {
	book := read(t, outline)

	want := []int{1, 2, 1}
	for i, level := range want {
		if book.Parts[i].Level != level {
			t.Errorf("%q is at level %d, want %d", book.Parts[i].Title, book.Parts[i].Level, level)
		}
	}
}

func TestAPartBeginsWhereItsPageBegins(t *testing.T) {
	book := read(t, outline)

	for _, part := range book.Parts {
		pos := -1
		for _, page := range book.Pages {
			if page.Offset == part.Offset {
				pos = page.Offset
			}
		}
		if pos < 0 {
			t.Errorf("%q begins at %d, which is not where any page begins", part.Title, part.Offset)
		}
	}
}

func TestAScanIsReadAndReturnsNothing(t *testing.T) {
	// A document with no text layer is not a failure. It is read, its pages are
	// counted, and what it says is nothing — which is what makes it a scan and
	// what a reader has to be able to tell.
	book := read(t, scan)

	if book.Text != "" {
		t.Errorf("text is %q, want it empty", book.Text)
	}
	if len(book.Pages) != 2 {
		t.Errorf("%d pages, want 2", len(book.Pages))
	}
}

func TestLocate(t *testing.T) {
	book := read(t, outline)

	// Keyed by a word in the text, so that the case says where it is asking
	// about and not what the offset happens to be.
	pos := func(word string) int {
		t.Helper()
		i := strings.Index(book.Text, word)
		if i < 0 {
			t.Fatalf("the fixture does not say %q", word)
		}
		return i
	}

	tests := []struct {
		name   string
		offset int
		part   string
		sheet  int
	}{
		{name: "before every name", offset: pos("Front"), part: "", sheet: 0},
		{name: "at a name", offset: pos("Opening"), part: "The First Part", sheet: 1},
		{name: "inside a nested name", offset: pos("closer"), part: "A Closer Reading", sheet: 2},
		{name: "after the last name", offset: pos("Afterword"), part: "Afterword", sheet: 3},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			where := book.Locate(c.offset)
			if where.Part != c.part {
				t.Errorf("part is %q, want %q", where.Part, c.part)
			}
			if where.Page != c.sheet {
				t.Errorf("page is %d, want %d", where.Page, c.sheet)
			}
		})
	}
}

func TestTheSameBytesGiveTheSameText(t *testing.T) {
	// A chunk keeps an offset and not the text, so a second reading of the same
	// file has to agree with the first.
	raw := fixture(t, outline)

	first, err := pdf.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := pdf.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	if first.Text != second.Text {
		t.Error("the same file read twice gives two texts")
	}
	if len(first.Parts) != len(second.Parts) {
		t.Fatalf("the same file names %d parts and then %d", len(first.Parts), len(second.Parts))
	}
	for i := range first.Parts {
		if first.Parts[i] != second.Parts[i] {
			t.Errorf("part %d is %+v and then %+v", i, first.Parts[i], second.Parts[i])
		}
	}
}

func TestWhatIsNotADocument(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want error
	}{
		{name: "prose", raw: fixture(t, prose), want: pdf.ErrNotPDF},
		{name: "nothing at all", raw: nil, want: pdf.ErrNotPDF},
		{name: "a header and no more", raw: []byte("%PDF-1.4\n"), want: pdf.ErrNotPDF},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			book, err := pdf.Read(c.raw)
			if !errors.Is(err, c.want) {
				t.Fatalf("error is %v, want %v", err, c.want)
			}
			if book != nil {
				t.Error("a file that is not a document was read into one")
			}
		})
	}
}

func TestReadingIsSafeFromSeveralGoroutines(t *testing.T) {
	// A vault is walked by one goroutine today and by more as soon as anything
	// asks it to be, and the library holds a document per worker.
	raw := fixture(t, tiny)

	isDone := make(chan string, 8)
	for i := 0; i < cap(isDone); i++ {
		go func() {
			book, err := pdf.Read(raw)
			if err != nil {
				isDone <- "error: " + err.Error()
				return
			}
			isDone <- book.Text
		}()
	}
	want := read(t, tiny).Text
	for i := 0; i < cap(isDone); i++ {
		if got := <-isDone; got != want {
			t.Errorf("one reading gave %q, want %q", got, want)
		}
	}
}

// A page is drawn up to a number of pixels and no further. The same page at a
// resolution that keeps it inside the bound is drawn.
func TestAPageTooLargeToDrawIsRefused(t *testing.T) {
	target, err := pdf.OpenReader(readerOf(fixture(t, vast)))
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	if _, err := target.RenderPage(0, 300); err == nil {
		t.Error("a page of 3.6 billion pixels was drawn")
	}
	rendered, err := target.RenderPage(0, 4)
	if err != nil {
		t.Fatalf("the page was not drawn at 4 DPI: %v", err)
	}
	if rendered.Bounds().Dx() < 700 {
		t.Errorf("the page is drawn %d across", rendered.Bounds().Dx())
	}
}

// A page says how big it is, and what it says is the size it is drawn at one of
// its own units to the pixel. A page drawn a quarter turn from the way its text
// is written is as wide as its text is high, and it says so.
func TestAPageReturnsHowBigItIs(t *testing.T) {
	for _, name := range []string{tiny, turned, labels} {
		t.Run(name, func(t *testing.T) {
			target, err := pdf.OpenReader(readerOf(fixture(t, name)))
			if err != nil {
				t.Fatal(err)
			}
			defer target.Close()

			for page := range target.GetPageCount() {
				width, height, err := target.GetPageSize(page)
				if err != nil {
					t.Fatalf("page %d: %v", page, err)
				}
				rendered, err := target.RenderPage(page, 72)
				if err != nil {
					t.Fatalf("page %d: %v", page, err)
				}
				rgba, ok := rendered.(*image.RGBA)
				if !ok {
					t.Fatalf("page %d: expected *image.RGBA, got %T", page, rendered)
				}
				hasPixels := false
				for _, b := range rgba.Pix {
					if b != 0 {
						hasPixels = true
						break
					}
				}
				if !hasPixels {
					t.Errorf("page %d: expected non-zero pixel data", page)
				}
				if got := float64(rendered.Bounds().Dx()); got < width-1 || got > width+1 {
					t.Errorf("page %d says it is %g across and is drawn %g", page, width, got)
				}
				if got := float64(rendered.Bounds().Dy()); got < height-1 || got > height+1 {
					t.Errorf("page %d says it is %g high and is drawn %g", page, height, got)
				}
			}
			if _, _, err := target.GetPageSize(-1); err == nil {
				t.Error("a negative page index was measured")
			}
			if _, _, err := target.GetPageSize(target.GetPageCount()); err == nil {
				t.Error("a page the document does not have was measured")
			}
			if _, err := target.RenderPage(-1, 72); err == nil {
				t.Error("a negative page index was rendered")
			}
			if _, err := target.RenderPage(target.GetPageCount(), 72); err == nil {
				t.Error("a page the document does not have was rendered")
			}
			drawnDefault, err := target.RenderPage(0, 0)
			if err != nil {
				t.Fatalf("render with 0 DPI failed: %v", err)
			}
			drawn300, err := target.RenderPage(0, 300)
			if err != nil {
				t.Fatalf("render with 300 DPI failed: %v", err)
			}
			if drawnDefault.Bounds() != drawn300.Bounds() {
				t.Errorf("page 0 default DPI bounds %v != 300 DPI bounds %v", drawnDefault.Bounds(), drawn300.Bounds())
			}
		})
	}
}

func TestOpenReaderInvalid(t *testing.T) {
	_, err := pdf.OpenReader(readerOf(fixture(t, prose)))
	if err == nil {
		t.Error("OpenReader succeeded on invalid bytes")
	}
}

// bytesFile is a document held in memory, with a Close that has nothing to release.
type bytesFile struct{ *bytes.Reader }

func (bytesFile) Close() error { return nil }

func readerOf(raw []byte) bytesFile { return bytesFile{bytes.NewReader(raw)} }
