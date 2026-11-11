package pdf

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cleanup"

	"github.com/klippa-app/go-pdfium/enums"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
	"github.com/tetratelabs/wazero"
)

// compilationDirVariable names the folder TestMain, in pdf_test.go, makes for the compiled PDFium of the
// test binary.
const compilationDirVariable = "NUMEN_PDF_TEST_COMPILATION_DIR"

// failOnReport fails the test with a background failure of the pool.
func failOnReport(tb testing.TB) func(string, error) {
	tb.Helper()
	return func(what string, err error) { tb.Errorf("%s: %v", what, err) }
}

func newCompilation(tb testing.TB) wazero.CompilationCache {
	tb.Helper()
	compilation, err := wazero.NewCompilationCacheWithDir(os.Getenv(compilationDirVariable))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := compilation.Close(context.WithoutCancel(tb.Context())); err != nil {
			tb.Error(err)
		}
	})
	return compilation
}

func newWorkers(tb testing.TB) *Workers {
	tb.Helper()
	workers := NewWorkers(Options{Report: failOnReport(tb), CompilationCache: newCompilation(tb), AfterFunc: nil, Workers: 0, IdleWorkers: 0})
	tb.Cleanup(func() {
		if err := workers.Close(); err != nil {
			tb.Error(err)
		}
	})
	return workers
}

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
// same walk, and a rectangle put against the wrong offset lands on the wrong
// word with nothing saying so.
func TestTheCharactersOfAPageAreItsText(t *testing.T) {
	for _, name := range []string{"tiny.pdf", "outline.pdf", "labels.pdf", "turned.pdf", "scan.pdf"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile("testdata/" + name)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := newWorkers(t).openBytes(raw)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup.Close(t, doc)

			for i := range doc.pages {
				response := doc.text(i)
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
				if joined.String() != response {
					t.Errorf("page %d: the characters say %q and the page says %q",
						i, joined.String(), response)
				}
			}
		})
	}
}

func where(t *testing.T, name string, pages ...int) (*Book, []HighlightBox) {
	t.Helper()
	return whereOn(t, newWorkers(t), name, pages...)
}

func whereOn(t *testing.T, workers *Workers, name string, pages ...int) (*Book, []HighlightBox) {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	book, err := workers.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	boxes, err := book.Highlights(workers, raw, pages)
	if err != nil {
		t.Fatal(err)
	}
	return book, boxes
}

// A box gives both the position of a text run on the page and the text run it
// covers, and the two refer to the same run.
func TestABoxCoversTheWordsItNames(t *testing.T) {
	book, boxes := where(t, "tiny.pdf", 0, 1)

	words := []string{"Alpha", "beta", "gamma", "Delta", "epsilon", "zeta"}
	pages := []int{0, 0, 0, 1, 1, 1}
	if len(boxes) != len(words) {
		t.Fatalf("%d boxes, want the %d words the document says", len(boxes), len(words))
	}
	for i, box := range boxes {
		if got := book.Text[box.Start:box.End]; got != words[i] {
			t.Errorf("box %d covers %q, want %q", i, got, words[i])
		}
		if box.Page != pages[i] {
			t.Errorf("%q is on page %d, want %d", words[i], box.Page, pages[i])
		}
	}
}

// Every box of every document covers one word: the offsets are in the text of
// the document as read, and a word is the unit the user sees.
func TestEveryBoxCoversOneWord(t *testing.T) {
	workers := newWorkers(t)
	for _, name := range []string{"tiny.pdf", "outline.pdf", "labels.pdf", "turned.pdf"} {
		t.Run(name, func(t *testing.T) {
			book, boxes := whereOn(t, workers, name, pagesOf(t, workers, name)...)
			if len(boxes) == 0 {
				t.Fatal("a document with a text layer lit nothing")
			}
			for _, box := range boxes {
				if box.Start < 0 || box.End > len(book.Text) {
					t.Fatalf("a box covers %d..%d of a text %d long",
						box.Start, box.End, len(book.Text))
				}
				word := book.Text[box.Start:box.End]
				if word == "" || strings.ContainsFunc(word, isSpace) {
					t.Errorf("a box covers %q, which is not one word", word)
				}
				if pos := book.Pages[box.Page].Offset; box.Start < pos {
					t.Errorf("a box on page %d covers %q, which is before that page", box.Page, word)
				}
			}
		})
	}
}

func isSpace(r rune) bool { return strings.ContainsRune(" \t\r\n\f", r) }

func pagesOf(t *testing.T, workers *Workers, name string) []int {
	t.Helper()
	book, _ := whereOn(t, workers, name)
	pages := make([]int, len(book.Pages))
	for i := range pages {
		pages[i] = i
	}
	return pages
}

func TestAScanLightsNothing(t *testing.T) {
	// A document with no text layer has no word positions, which is what a scan
	// is.
	_, boxes := where(t, "scan.pdf", 0, 1)

	if len(boxes) != 0 {
		t.Errorf("a scan lit %d boxes", len(boxes))
	}
}

func TestARectangleIsAFractionOfThePage(t *testing.T) {
	workers := newWorkers(t)
	// A page drawn at any size lines up by multiplying, so a rectangle is
	// somewhere between nothing and the whole page.
	for _, name := range []string{"tiny.pdf", "outline.pdf", "labels.pdf", "turned.pdf"} {
		t.Run(name, func(t *testing.T) {
			_, boxes := whereOn(t, workers, name, pagesOf(t, workers, name)...)
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
		if boxes[i].Start < boxes[i-1].Start {
			t.Fatalf("box %d begins at %d, after one beginning at %d",
				i, boxes[i].Start, boxes[i-1].Start)
		}
	}
}

// One page is highlighted on its own. A document is highlighted a few pages at
// a time, and pages that were not requested are not read.
func TestOnePageIsLitAndTheRestAreNot(t *testing.T) {
	book, boxes := where(t, "outline.pdf", 2)

	if len(boxes) == 0 {
		t.Fatal("the page lit nothing")
	}
	from, to := book.Pages[2].Offset, book.Pages[3].Offset
	for _, box := range boxes {
		if box.Page != 2 {
			t.Errorf("unexpected highlight on page %d", box.Page)
		}
		if box.Start < from || box.Start >= to {
			t.Errorf("a box begins at %d, outside the page's %d..%d", box.Start, from, to)
		}
	}
}

func TestPagesTheDocumentDoesNotHaveAreLitNowhere(t *testing.T) {
	_, boxes := where(t, "tiny.pdf", -1, 7)

	if len(boxes) != 0 {
		t.Errorf("%d boxes for pages the document does not have", len(boxes))
	}
}

// A page is rendered rotated by its own rotation, and a box is a fraction of
// the page as rendered. A line of a rotated page runs down it.
func TestATurnedPageIsLitTheWayItIsGenerated(t *testing.T) {
	book, boxes := where(t, "turned.pdf", 0)

	if len(boxes) != 3 {
		t.Fatalf("%d boxes, want the three words the page says", len(boxes))
	}
	for _, box := range boxes {
		word := book.Text[box.Start:box.End]
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
	mark := responses.CharPosition{Left: 25, Right: 30, Top: 25, Bottom: 5} // gap = 5 == 5, shared = 15 >= 10
	if !isWordContinued(w, mark) {
		t.Error("expected word continued for adjacent char at exact wordGap boundary")
	}

	// Exact shared height boundary: shared = 10 == 0.5 * 20
	exactShared := responses.CharPosition{Left: 22, Right: 30, Top: 20, Bottom: 0}
	if !isWordContinued(w, exactShared) {
		t.Error("expected word continued at exact sameLine shared height boundary")
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

	// 4. Zero or negative height
	zeroHigh := responses.CharPosition{Left: 22, Right: 30, Top: 10, Bottom: 10}
	if isWordContinued(w, zeroHigh) {
		t.Error("expected word NOT continued for zero height char")
	}
	negHigh := responses.CharPosition{Left: 22, Right: 30, Top: 5, Bottom: 10}
	if isWordContinued(w, negHigh) {
		t.Error("expected word NOT continued for negative height char")
	}
	if isWordContinued(zeroHigh, mark) {
		t.Error("expected word NOT continued when word has zero height")
	}

	// 5. Backwards or equal Right: mark.Right <= word.Right
	backwards := responses.CharPosition{Left: 15, Right: 20, Top: 30, Bottom: 10}
	if isWordContinued(w, backwards) {
		t.Error("expected word NOT continued when char Right equals word Right")
	}
}

func TestWordReadingAddAndFinish(t *testing.T) {
	sheet := pageGeometry{width: 100, height: 100, rotation: 0}
	r := &wordReading{sheet: sheet, index: 0}
	r.finish()
	if len(r.boxes) != 0 {
		t.Fatalf("expected 0 boxes after finishing empty wordReading, got %d", len(r.boxes))
	}

	validMark := responses.CharPosition{Left: 10, Right: 20, Top: 30, Bottom: 10}
	r.add(0, 1, validMark, false)
	// Degenerate width mark finishes the current word
	r.add(1, 1, responses.CharPosition{Left: 20, Right: 20, Top: 30, Bottom: 10}, false)
	if len(r.boxes) != 1 || r.length != 0 {
		t.Fatalf("expected 1 box and length 0 after degenerate width mark, got %d boxes, length %d", len(r.boxes), r.length)
	}

	r.add(2, 1, validMark, false)
	// Negative width mark finishes the current word
	r.add(3, 1, responses.CharPosition{Left: 25, Right: 20, Top: 30, Bottom: 10}, false)
	if len(r.boxes) != 2 || r.length != 0 {
		t.Fatalf("expected 2 boxes and length 0 after negative width mark, got %d boxes, length %d", len(r.boxes), r.length)
	}

	r.add(4, 1, validMark, false)
	// Degenerate height mark finishes the current word
	r.add(5, 1, responses.CharPosition{Left: 10, Right: 20, Top: 10, Bottom: 10}, false)
	if len(r.boxes) != 3 || r.length != 0 {
		t.Fatalf("expected 3 boxes and length 0 after degenerate height mark, got %d boxes, length %d", len(r.boxes), r.length)
	}

	r.add(6, 1, validMark, false)
	// Negative height mark finishes the current word
	r.add(7, 1, responses.CharPosition{Left: 10, Right: 20, Top: 5, Bottom: 10}, false)
	if len(r.boxes) != 4 || r.length != 0 {
		t.Fatalf("expected 4 boxes and length 0 after negative height mark, got %d boxes, length %d", len(r.boxes), r.length)
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
	workers := newWorkers(t)
	book, err := workers.Read(raw)
	if err != nil {
		t.Fatal(err)
	}

	// Empty wanted pages
	boxes, err := book.Highlights(workers, raw, nil)
	if err != nil || boxes != nil {
		t.Errorf("Highlights(nil) got %v, err %v; want nil, nil", boxes, err)
	}
	boxes, err = book.Highlights(workers, raw, []int{})
	if err != nil || boxes != nil {
		t.Errorf("Highlights([]) got %v, err %v; want nil, nil", boxes, err)
	}

	// Invalid PDF bytes
	_, err = book.Highlights(workers, []byte("not a pdf"), []int{0})
	if err == nil {
		t.Error("expected error for invalid PDF bytes in Highlights")
	}
}

func TestDocumentWordsInvalidPage(t *testing.T) {
	doc, err := newWorkers(t).openBytes(fixture(t, "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, doc)

	if boxes := doc.words(9999, 0); len(boxes) != 0 {
		t.Errorf("doc.words(9999) returned %d boxes, want 0", len(boxes))
	}
}

type mockPageWorker struct {
	width    float64
	height   float64
	rotation int
}

func (m *mockPageWorker) GetPageSize(_ *requests.GetPageSize) (*responses.GetPageSize, error) {
	return &responses.GetPageSize{Width: m.width, Height: m.height}, nil
}

func (m *mockPageWorker) GetPageRotation(_ *requests.FPDFPage_GetRotation) (*responses.FPDFPage_GetRotation, error) {
	return &responses.FPDFPage_GetRotation{PageRotation: enums.FPDF_PAGE_ROTATION(m.rotation)}, nil
}

func TestGetSheetRejectsZeroDimensions(t *testing.T) {
	docZeroWidth := &mockPageWorker{width: 0, height: 100}
	if _, ok := getSheet(docZeroWidth, requests.Page{}); ok {
		t.Error("expected getSheet to reject zero width")
	}
	docOneWidth := &mockPageWorker{width: 1, height: 100}
	if _, ok := getSheet(docOneWidth, requests.Page{}); !ok {
		t.Error("expected getSheet to accept width 1")
	}

	docZeroHeight := &mockPageWorker{width: 100, height: 0}
	if _, ok := getSheet(docZeroHeight, requests.Page{}); ok {
		t.Error("expected getSheet to reject zero height")
	}
	docOneHeight := &mockPageWorker{width: 100, height: 1}
	if _, ok := getSheet(docOneHeight, requests.Page{}); !ok {
		t.Error("expected getSheet to accept height 1")
	}

	docValid := &mockPageWorker{width: 100, height: 200, rotation: 0}
	sheet, ok := getSheet(docValid, requests.Page{})
	if !ok || sheet.width != 100 || sheet.height != 200 {
		t.Errorf("expected valid sheet (100, 200), got %+v, %v", sheet, ok)
	}

	for _, rot := range []int{1, 3} {
		turnedSheet, ok := getSheet(&mockPageWorker{width: 100, height: 200, rotation: rot}, requests.Page{})
		if !ok || turnedSheet.width != 200 || turnedSheet.height != 100 {
			t.Errorf("rot %d: expected swapped sheet (200, 100), got %+v, %v", rot, turnedSheet, ok)
		}
	}
	rot2Sheet, ok := getSheet(&mockPageWorker{width: 100, height: 200, rotation: 2}, requests.Page{})
	if !ok || rot2Sheet.width != 100 || rot2Sheet.height != 200 {
		t.Errorf("rot 2: expected unswapped sheet (100, 200), got %+v, %v", rot2Sheet, ok)
	}
}
