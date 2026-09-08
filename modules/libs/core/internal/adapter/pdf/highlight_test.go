package pdf

import (
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/highlight"
	"github.com/klippa-app/go-pdfium/requests"
)

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
			doc, err := open(raw)
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
func where(t *testing.T, name string, pages ...int) (*Book, []highlight.Box) {
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
