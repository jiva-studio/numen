package pdf

import (
	"os"
	"strings"
	"testing"

	pdfium "github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// A page's text and the characters it is made of are one text.
//
// Where a word sits on the page is asked for character by character, and where a
// chunk sits in the document is an offset into the page's text. The two are the
// same walk, and a rectangle put against the wrong offset lands on the wrong
// word with nothing saying so.
func TestTheCharactersOfAPageAreItsText(t *testing.T) {
	for _, name := range []string{"tiny.pdf", "outline.pdf", "labels.pdf", "turned.pdf", "scan.pdf"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile("testdata/" + name)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := openBytes(raw)
			if err != nil {
				t.Fatal(err)
			}
			defer doc.close()

			for i := 0; i < doc.pages; i++ {
				said := doc.text(i)
				read, err := doc.worker.GetPageTextStructured(&requests.GetPageTextStructured{
					Page: requests.Page{ByIndex: &requests.PageByIndex{Document: doc.ref, Index: i}},
					Mode: requests.GetPageTextStructuredModeChars,
				})
				if err != nil {
					t.Fatalf("page %d: %v", i, err)
				}
				var joined strings.Builder
				for _, one := range read.Chars {
					joined.WriteString(one.Text)
				}
				if joined.String() != said {
					t.Errorf("page %d: the characters say %q and the page says %q",
						i, joined.String(), said)
				}
			}
		})
	}
}

// where reads a fixture and says where the words of some of its pages are.
func where(t *testing.T, name string, pages ...int) (*Book, []HighlightBox) {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	book, err := Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	boxes, err := book.Highlights(raw, pages)
	if err != nil {
		t.Fatal(err)
	}
	return book, boxes
}

// A box says both where a run of the text is on the page and which run of the
// text it is, and the two are the same run.
func TestABoxCoversTheWordsItNames(t *testing.T) {
	book, boxes := where(t, "tiny.pdf", 0, 1)

	words := []string{"Alpha", "beta", "gamma", "Delta", "epsilon", "zeta"}
	pages := []int{0, 0, 0, 1, 1, 1}
	if len(boxes) != len(words) {
		t.Fatalf("%d boxes, want the %d words the document says", len(boxes), len(words))
	}
	for i, box := range boxes {
		if got := book.Text[box.From:box.To]; got != words[i] {
			t.Errorf("box %d covers %q, want %q", i, got, words[i])
		}
		if box.Page != pages[i] {
			t.Errorf("%q is on page %d, want %d", words[i], box.Page, pages[i])
		}
	}
}

// Every box of every document covers one word: the offsets are in the text the
// document was read into, and a word is what a person sees.
func TestEveryBoxCoversOneWord(t *testing.T) {
	for _, name := range []string{"tiny.pdf", "outline.pdf", "labels.pdf", "turned.pdf"} {
		t.Run(name, func(t *testing.T) {
			book, boxes := where(t, name, pagesOf(t, name)...)
			if len(boxes) == 0 {
				t.Fatal("a document with a text layer lit nothing")
			}
			for _, box := range boxes {
				if box.From < 0 || box.To > len(book.Text) {
					t.Fatalf("a box covers %d..%d of a text %d long",
						box.From, box.To, len(book.Text))
				}
				word := book.Text[box.From:box.To]
				if word == "" || strings.ContainsFunc(word, isSpace) {
					t.Errorf("a box covers %q, which is not one word", word)
				}
				if at := book.Pages[box.Page].Offset; box.From < at {
					t.Errorf("a box on page %d covers %q, which is before that page", box.Page, word)
				}
			}
		})
	}
}

func isSpace(r rune) bool { return strings.ContainsRune(" \t\r\n\f", r) }

// pagesOf is every page of a fixture.
func pagesOf(t *testing.T, name string) []int {
	t.Helper()
	book, _ := where(t, name)
	pages := make([]int, len(book.Pages))
	for i := range pages {
		pages[i] = i
	}
	return pages
}

func TestAScanLightsNothing(t *testing.T) {
	// A document with no text layer says nothing about where its words are,
	// and that is the whole of what a scan is.
	_, boxes := where(t, "scan.pdf", 0, 1)

	if len(boxes) != 0 {
		t.Errorf("a scan lit %d boxes", len(boxes))
	}
}

func TestARectangleIsAFractionOfThePage(t *testing.T) {
	// A page drawn at any size lines up by multiplying, so a rectangle is
	// somewhere between nothing and the whole page.
	for _, name := range []string{"tiny.pdf", "outline.pdf", "labels.pdf", "turned.pdf"} {
		t.Run(name, func(t *testing.T) {
			_, boxes := where(t, name, pagesOf(t, name)...)
			for _, box := range boxes {
				if box.MinX < 0 || box.MinY < 0 || box.MaxX > 1 || box.MaxY > 1 {
					t.Errorf("a box covers %v, which is off the page", box)
				}
				if box.MinX >= box.MaxX || box.MinY >= box.MaxY {
					t.Errorf("a box covers %v, which is nothing at all", box)
				}
			}
		})
	}
}

func TestBoxesRiseInOrder(t *testing.T) {
	// A run of the text is found by halving the boxes, which holds only while
	// they ascend.
	book, boxes := where(t, "outline.pdf", 3, 0, 2, 1)

	if len(book.Pages) != 4 || len(boxes) == 0 {
		t.Fatalf("%d boxes over %d pages", len(boxes), len(book.Pages))
	}
	for i := 1; i < len(boxes); i++ {
		if boxes[i].From < boxes[i-1].From {
			t.Fatalf("box %d begins at %d, after one beginning at %d",
				i, boxes[i].From, boxes[i-1].From)
		}
	}
}

// One page is lit on its own. A document is lit a few pages at a time, and the
// pages nobody asked about are not read.
func TestOnePageIsLitAndTheRestAreNot(t *testing.T) {
	book, boxes := where(t, "outline.pdf", 2)

	if len(boxes) == 0 {
		t.Fatal("the page lit nothing")
	}
	from, to := book.Pages[2].Offset, book.Pages[3].Offset
	for _, box := range boxes {
		if box.Page != 2 {
			t.Errorf("page %d was lit and nobody asked about it", box.Page)
		}
		if box.From < from || box.From >= to {
			t.Errorf("a box begins at %d, outside the page's %d..%d", box.From, from, to)
		}
	}
}

func TestPagesTheDocumentDoesNotHaveAreLitNowhere(t *testing.T) {
	_, boxes := where(t, "tiny.pdf", -1, 7)

	if len(boxes) != 0 {
		t.Errorf("%d boxes for pages the document does not have", len(boxes))
	}
}

// A page is drawn turned by however much it asks to be, and a box is a fraction
// of the page as it is drawn. A line of a turned page runs down it.
func TestATurnedPageIsLitTheWayItIsDrawn(t *testing.T) {
	book, boxes := where(t, "turned.pdf", 0)

	if len(boxes) != 3 {
		t.Fatalf("%d boxes, want the three words the page says", len(boxes))
	}
	for _, box := range boxes {
		word := book.Text[box.From:box.To]
		if box.MaxY-box.MinY <= box.MaxX-box.MinX {
			t.Errorf("%q covers %v, which lies across the page", word, box)
		}
	}
	if first, last := boxes[0], boxes[2]; first.MinY >= last.MinY {
		t.Errorf("the line begins at %v and ends at %v", first, last)
	}
}

func TestIsWordContinued(t *testing.T) {
	w := responses.CharPosition{Left: 10, Right: 20, Top: 30, Bottom: 10} // width 10, height 20
	// 1. Same line, normal gap (high = 20, wordGap * 20 = 5)
	mark := responses.CharPosition{Left: 22, Right: 30, Top: 30, Bottom: 10} // gap 2 <= 5
	if !isWordContinued(w, mark) {
		t.Error("expected word continued for adjacent char")
	}

	// 2. Gap too wide: gap 6 > 5
	tooFar := responses.CharPosition{Left: 26, Right: 35, Top: 30, Bottom: 10}
	if isWordContinued(w, tooFar) {
		t.Error("expected word NOT continued when gap exceeds wordGap*high")
	}

	// 3. Different vertical line: shared height < 0.5 * 20 = 10
	diffLine := responses.CharPosition{Left: 22, Right: 30, Top: 15, Bottom: -5}
	if isWordContinued(w, diffLine) {
		t.Error("expected word NOT continued when characters share less than sameLine of height")
	}

	// 4. Zero height
	zeroHigh := responses.CharPosition{Left: 22, Right: 30, Top: 10, Bottom: 10}
	if isWordContinued(w, zeroHigh) {
		t.Error("expected word NOT continued for zero height char")
	}

	// 5. Backwards: mark.Right <= word.Right
	backwards := responses.CharPosition{Left: 15, Right: 19, Top: 30, Bottom: 10}
	if isWordContinued(w, backwards) {
		t.Error("expected word NOT continued when char is backwards")
	}
}

func TestGetOrderedPages(t *testing.T) {
	got := getOrderedPages([]int{2, -1, 0, 0, 1}, 2)
	want := []int{0, 1}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("getOrderedPages got %v, want %v", got, want)
	}

	// Boundary: page == most should be excluded
	if got := getOrderedPages([]int{2}, 2); len(got) != 0 {
		t.Errorf("expected page >= most to be excluded, got %v", got)
	}
}

func TestClampToPage(t *testing.T) {
	if got := clampToPage(-0.5); got != 0 {
		t.Errorf("clampToPage(-0.5) = %g, want 0", got)
	}
	if got := clampToPage(1.5); got != 1 {
		t.Errorf("clampToPage(1.5) = %g, want 1", got)
	}
	if got := clampToPage(0.5); got != 0.5 {
		t.Errorf("clampToPage(0.5) = %g, want 0.5", got)
	}
}

func TestGetPagePointRotations(t *testing.T) {
	// width 100, height 200
	// Rotation 0: default -> y = p.height - y
	p0 := pageGeometry{width: 100, height: 200, rotation: 0}
	x, y := p0.getPagePoint(10, 20)
	if x != 0.1 || y != 0.9 {
		t.Errorf("rot 0: got (%g, %g), want (0.1, 0.9)", x, y)
	}

	// Rotation 1: 90 deg -> x, y, wide, high = y, x, high, wide
	p1 := pageGeometry{width: 100, height: 200, rotation: 1}
	x, y = p1.getPagePoint(10, 20)
	if x != 0.1 || y != 0.1 {
		t.Errorf("rot 1: got (%g, %g), want (0.1, 0.1)", x, y)
	}

	// Rotation 2: 180 deg -> x = width - x
	p2 := pageGeometry{width: 100, height: 200, rotation: 2}
	x, y = p2.getPagePoint(10, 20)
	if x != 0.9 || y != 0.1 {
		t.Errorf("rot 2: got (%g, %g), want (0.9, 0.1)", x, y)
	}

	// Rotation 3: 270 deg -> x, y, wide, high = height - y, width - x, high, wide
	p3 := pageGeometry{width: 100, height: 200, rotation: 3}
	x, y = p3.getPagePoint(10, 20)
	if x != 0.9 || y != 0.9 {
		t.Errorf("rot 3: got (%g, %g), want (0.9, 0.9)", x, y)
	}
}

func TestHighlightsEmptyPagesOrInvalid(t *testing.T) {
	raw := fixture(t, "tiny.pdf")
	book, err := Read(raw)
	if err != nil {
		t.Fatal(err)
	}

	// Empty wanted pages
	boxes, err := book.Highlights(raw, nil)
	if err != nil || boxes != nil {
		t.Errorf("Highlights(nil) got %v, err %v; want nil, nil", boxes, err)
	}
	boxes, err = book.Highlights(raw, []int{})
	if err != nil || boxes != nil {
		t.Errorf("Highlights([]) got %v, err %v; want nil, nil", boxes, err)
	}

	// Invalid PDF bytes
	_, err = book.Highlights([]byte("not a pdf"), []int{0})
	if err == nil {
		t.Error("expected error for invalid PDF bytes in Highlights")
	}
}

func TestDocumentWordsInvalidPage(t *testing.T) {
	doc, err := openBytes(fixture(t, "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.close()

	if boxes := doc.words(9999, 0); len(boxes) != 0 {
		t.Errorf("doc.words(9999) returned %d boxes, want 0", len(boxes))
	}
}

type mockPageWorker struct {
	pdfium.Pdfium
	width  float64
	height float64
}

func (m *mockPageWorker) GetPageSize(req *requests.GetPageSize) (*responses.GetPageSize, error) {
	return &responses.GetPageSize{Width: m.width, Height: m.height}, nil
}

func (m *mockPageWorker) FPDFPage_GetRotation(req *requests.FPDFPage_GetRotation) (*responses.FPDFPage_GetRotation, error) {
	return &responses.FPDFPage_GetRotation{}, nil
}

func TestGetSheetRejectsZeroDimensions(t *testing.T) {
	docZeroWidth := &document{worker: &mockPageWorker{width: 0, height: 100}}
	if _, ok := docZeroWidth.getSheet(requests.Page{}); ok {
		t.Error("expected getSheet to reject zero width")
	}

	docZeroHeight := &document{worker: &mockPageWorker{width: 100, height: 0}}
	if _, ok := docZeroHeight.getSheet(requests.Page{}); ok {
		t.Error("expected getSheet to reject zero height")
	}

	docValid := &document{worker: &mockPageWorker{width: 100, height: 200}}
	sheet, ok := docValid.getSheet(requests.Page{})
	if !ok || sheet.width != 100 || sheet.height != 200 {
		t.Errorf("expected valid sheet (100, 200), got %+v, %v", sheet, ok)
	}
}
