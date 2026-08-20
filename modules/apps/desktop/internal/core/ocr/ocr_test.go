package ocr_test

import (
	"image"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
)

// line is one run of words at a place on the page, written the way a detector
// hands it over.
func line(x0, y0, x1, y1 int, text string) ocr.Line {
	return ocr.Line{Box: image.Rect(x0, y0, x1, y1), Text: text, Score: 1}
}

func TestLinesAreGroupedByWhatTheyShareVertically(t *testing.T) {
	// Two runs on one line, and a third beneath them. The one beneath is a line
	// of its own however far left it starts.
	got, _ := ocr.Assemble([]ocr.Line{
		line(10, 100, 60, 120, "second"),
		line(0, 98, 8, 121, "the"),
		line(0, 140, 90, 160, "line beneath"),
	})
	want := "the second line beneath"
	if got != want {
		t.Errorf("assembled %q, want %q", got, want)
	}
}

func TestALineBrokenByAHyphenContinues(t *testing.T) {
	tests := []struct {
		name string
		ends string
		next string
		want string
	}{
		{name: "a hyphen joins", ends: "under-", next: "standing follows", want: "understanding follows"},
		{name: "a soft hyphen joins", ends: "under­", next: "standing follows", want: "understanding follows"},
		{name: "a dash between words does not", ends: "a word -", next: "and another", want: "a word - and another"},
		{name: "nothing to join", ends: "a line", next: "another line", want: "a line another line"},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			got, _ := ocr.Assemble([]ocr.Line{
				line(0, 0, 100, 20, c.ends),
				line(0, 40, 100, 60, c.next),
			})
			if got != c.want {
				t.Errorf("assembled %q, want %q", got, c.want)
			}
		})
	}
}

func TestARegionFoundTwiceIsReadOnce(t *testing.T) {
	// The same paragraph, found twice at slightly different bounds, and a
	// heading inside the second of them. Reading all three would say the
	// paragraph twice and the heading twice.
	regions := []ocr.Region{
		{Label: "text", Score: 0.9, Rect: image.Rect(0, 0, 100, 100), Order: 1},
		{Label: "text", Score: 0.8, Rect: image.Rect(2, 1, 99, 101), Order: 2},
		{Label: "paragraph_title", Score: 0.7, Rect: image.Rect(10, 10, 90, 20), Order: 3},
		{Label: "text", Score: 0.95, Rect: image.Rect(0, 200, 100, 300), Order: 4},
	}
	kept := ocr.Distinct(regions, 0.6)

	if len(kept) != 2 {
		var got []string
		for _, r := range kept {
			got = append(got, r.Label)
		}
		t.Fatalf("kept %d regions (%s), want the two that do not cover each other", len(kept), strings.Join(got, ", "))
	}
	if kept[0].Order != 1 || kept[1].Order != 4 {
		t.Errorf("kept the regions at %d and %d, want 1 and 4", kept[0].Order, kept[1].Order)
	}
}

func TestRegionsComeBackInReadingOrder(t *testing.T) {
	// The model answers in whatever order it found them, and says where each
	// belongs. Sorting on that is what keeps the second column after the first.
	kept := ocr.Distinct([]ocr.Region{
		{Label: "text", Score: 0.9, Rect: image.Rect(200, 0, 300, 100), Order: 3},
		{Label: "text", Score: 0.9, Rect: image.Rect(0, 0, 100, 100), Order: 1},
		{Label: "text", Score: 0.9, Rect: image.Rect(0, 200, 100, 300), Order: 2},
	}, 0.6)

	for i, want := range []int{1, 2, 3} {
		if kept[i].Order != want {
			t.Errorf("region %d is at %d, want %d", i, kept[i].Order, want)
		}
	}
}

func TestAnArtifactSaysWhatEachPageSays(t *testing.T) {
	pages := []ocr.Page{
		{Label: "i", Blocks: []ocr.Block{{Label: "text", Text: "Preface."}}},
		{Label: "1", Blocks: []ocr.Block{
			{Label: "paragraph_title", Text: "THE FIRST PART"},
			{Label: "text", Text: "The body begins."},
		}},
	}
	raw, _ := ocr.Write(pages)

	text, marks := ocr.Read(raw)
	if len(marks) != 2 {
		t.Fatalf("read %d pages, want 2", len(marks))
	}
	for i, want := range []string{"i", "1"} {
		if marks[i].Label != want {
			t.Errorf("page %d is called %q, want %q", i, marks[i].Label, want)
		}
	}

	// The marks are not in the text: an offset in it is an offset in the prose.
	if strings.ContainsRune(text, '\x0c') {
		t.Errorf("the text still carries a page mark: %q", text)
	}
	if !strings.HasPrefix(text[marks[1].Offset:], "THE FIRST PART") {
		t.Errorf("the second page begins %q", text[marks[1].Offset:])
	}
	for _, want := range []string{"Preface.", "THE FIRST PART", "The body begins."} {
		if !strings.Contains(text, want) {
			t.Errorf("the artifact does not say %q", want)
		}
	}
}

func TestAnArtifactWithNoMarksIsAllProse(t *testing.T) {
	// A file the person wrote, or one written by something else. Dropping its
	// text because it names no pages would lose the whole document.
	text, marks := ocr.Read([]byte("just words, and no pages named"))

	if len(marks) != 0 {
		t.Errorf("found %d pages in a file that names none", len(marks))
	}
	if text != "just words, and no pages named" {
		t.Errorf("read %q", text)
	}
}

func TestAPageWithNothingOnItIsStillAPage(t *testing.T) {
	raw, _ := ocr.Write([]ocr.Page{
		{Label: "1", Blocks: []ocr.Block{{Label: "text", Text: "Something."}}},
		{Label: "2"},
		{Label: "3", Blocks: []ocr.Block{{Label: "text", Text: "Something else."}}},
	})

	_, marks := ocr.Read(raw)
	if len(marks) != 3 {
		t.Fatalf("read %d pages, want 3 — a blank page is a fact about the document", len(marks))
	}
	if marks[2].Label != "3" {
		t.Errorf("the third page is called %q", marks[2].Label)
	}
}

func TestWritingAndReadingAgreeAboutEveryOffset(t *testing.T) {
	pages := []ocr.Page{
		{Label: "i", Blocks: []ocr.Block{{Label: "text", Text: "Alpha."}}},
		{Label: "ii", Blocks: []ocr.Block{{Label: "text", Text: "Beta."}, {Label: "text", Text: "Gamma."}}},
		{Label: "1", Blocks: []ocr.Block{{Label: "text", Text: "Delta."}}},
	}
	raw, _ := ocr.Write(pages)
	text, marks := ocr.Read(raw)

	// Every mark is inside the text and they ascend, which is what Locate
	// searches through.
	for i, mark := range marks {
		if mark.Offset < 0 || mark.Offset > len(text) {
			t.Errorf("page %q begins at %d, and the text is %d long", mark.Label, mark.Offset, len(text))
		}
		if i > 0 && mark.Offset < marks[i-1].Offset {
			t.Errorf("page %q begins before the page before it", mark.Label)
		}
	}
	// The first page's prose is where it says it is.
	if got := strings.TrimSpace(text[marks[0].Offset:marks[1].Offset]); got != "Alpha." {
		t.Errorf("the first page says %q, want %q", got, "Alpha.")
	}
}

func TestABoxReadAsNothingIsNotAWord(t *testing.T) {
	// A detector finds a run of words and the recogniser reads nothing in it.
	// Joining that in puts two spaces where the page prints one.
	got, _ := ocr.Assemble([]ocr.Line{
		line(0, 0, 30, 20, "As the"),
		line(35, 0, 40, 20, ""),
		line(45, 0, 90, 20, "Lord traveled"),
		line(95, 0, 99, 20, "   "),
		line(105, 0, 150, 20, "from Puri"),
	})
	want := "As the Lord traveled from Puri"
	if got != want {
		t.Errorf("assembled %q, want %q", got, want)
	}
}

func TestALineWrittenForTheWriterIsNotProse(t *testing.T) {
	// A run stopped part way writes down how far it got. That line is not what
	// the page says, and an offset into the prose must not count it.
	first, _ := ocr.Write([]ocr.Page{
		{Label: "1", Blocks: []ocr.Block{{Label: "text", Text: "Alpha."}}},
	})
	second, _ := ocr.Write([]ocr.Page{
		{Label: "2", Blocks: []ocr.Block{{Label: "text", Text: "Beta."}}},
	})
	raw := append([]byte("\x00pages 2\n"), first...)
	raw = append(raw, []byte("\x00pages 4\n")...)
	raw = append(raw, second...)

	text, marks := ocr.Read(raw)
	if strings.Contains(text, "pages") {
		t.Errorf("the prose carries a note meant for the writer: %q", text)
	}
	if len(marks) != 2 {
		t.Fatalf("read %d pages, want 2", len(marks))
	}
	if got := strings.TrimSpace(text[marks[1].Offset:]); got != "Beta." {
		t.Errorf("the second page says %q, want %q", got, "Beta.")
	}
}

func TestEveryBoxSaysWhereItsWordsAreInTheProse(t *testing.T) {
	// Two pages, and the first of them two regions. The offsets are in the
	// prose, so the second page's boxes are past everything the first says.
	first, firstSpans := ocr.Assemble([]ocr.Line{
		line(0, 0, 50, 20, "Alpha"),
		line(60, 0, 100, 20, "beta"),
		line(0, 40, 60, 60, "gamma"),
	})
	last, lastSpans := ocr.Assemble([]ocr.Line{
		line(0, 0, 70, 20, "Epsilon"),
		line(80, 0, 120, 20, "zeta"),
	})
	// The first page prints its number above what it says. That region is not
	// prose, and the words under it are where the prose puts them.
	number, opening := ocr.Number([]ocr.Block{
		{
			Label: "number",
			Text:  "2",
			Place: true,
			Spans: []ocr.Span{{Box: image.Rect(300, 0, 320, 10), Length: 1}},
		},
		{Label: "text", Text: first, Spans: firstSpans},
		// A region read by something that reports no rectangles. It says
		// what it says and the prose after it moves along by that much.
		{Label: "text", Text: "Delta."},
	})
	pages := []ocr.Page{
		{At: 0, Number: number, Size: image.Pt(600, 800), Blocks: opening},
		// A page printing nothing, in a document that calls it something.
		{At: 1, Label: "1", Size: image.Pt(600, 800), Blocks: []ocr.Block{
			{Label: "text", Text: last, Spans: lastSpans},
		}},
	}
	raw, boxes := ocr.Write(pages)
	text, marks := ocr.Read(raw)

	for i, want := range []string{"2", "1"} {
		if marks[i].Label != want {
			t.Errorf("page %d is called %q, want %q", i, marks[i].Label, want)
		}
	}

	want := []string{"Alpha", "beta", "gamma", "Epsilon", "zeta"}
	if len(boxes) != len(want) {
		t.Fatalf("wrote %d boxes, want %d", len(boxes), len(want))
	}
	for i, box := range boxes {
		if box.Start < 0 || box.Start+box.Length > len(text) {
			t.Fatalf("box %d covers %d..%d, and the prose is %d long", i, box.Start, box.Start+box.Length, len(text))
		}
		if got := text[box.Start : box.Start+box.Length]; got != want[i] {
			t.Errorf("box %d reads %q, want %q", i, got, want[i])
		}
	}

	if boxes[3].Page != 1 {
		t.Errorf("the fourth box is on page %d, want 1", boxes[3].Page)
	}
	// The rectangle is the share of the page the box covers.
	if boxes[0].MinX != 0 || boxes[0].MaxX != 50.0/600 || boxes[0].MaxY != 20.0/800 {
		t.Errorf("the first box covers %v..%v, %v..%v", boxes[0].MinX, boxes[0].MaxX, boxes[0].MinY, boxes[0].MaxY)
	}
	for i, box := range boxes {
		for _, at := range []float32{box.MinX, box.MinY, box.MaxX, box.MaxY} {
			if at < 0 || at > 1 {
				t.Errorf("box %d reaches %v, and the page is one wide and one high", i, at)
			}
		}
	}
}

func TestAJoinedWordLeavesTheHyphenBoxOneByteShorter(t *testing.T) {
	// The hyphen is gone from the end of the box that carried it, so that box
	// covers the first half of the word and no more.
	text, spans := ocr.Assemble([]ocr.Line{
		line(0, 0, 100, 20, "under-"),
		line(0, 40, 200, 60, "standing follows"),
	})
	if text != "understanding follows" {
		t.Fatalf("assembled %q", text)
	}
	if len(spans) != 2 {
		t.Fatalf("wrote %d spans, want 2", len(spans))
	}
	if got := text[spans[0].Start : spans[0].Start+spans[0].Length]; got != "under" {
		t.Errorf("the first box reads %q, want %q", got, "under")
	}
	if got := text[spans[1].Start : spans[1].Start+spans[1].Length]; got != "standing follows" {
		t.Errorf("the second box reads %q, want %q", got, "standing follows")
	}
}

func TestAPageNothingWasMeasuredOnHasNoBoxes(t *testing.T) {
	// A page with no size gives no fraction of itself to divide a rectangle by.
	raw, boxes := ocr.Write([]ocr.Page{
		{At: 0, Label: "1", Blocks: []ocr.Block{{
			Label: "text",
			Text:  "Alpha beta",
			Spans: []ocr.Span{{Box: image.Rect(0, 0, 50, 20), Start: 0, Length: 5}},
		}}},
	})

	if len(boxes) != 0 {
		t.Errorf("wrote %d boxes for a page nothing was measured on", len(boxes))
	}
	text, marks := ocr.Read(raw)
	if len(marks) != 1 || !strings.Contains(text, "Alpha beta") {
		t.Errorf("the page says %q", text)
	}
}

func TestAJoinedWordLeavesTheHyphenBoxShorterByTheHyphen(t *testing.T) {
	// A hyphen is one byte, or two, or three. What the box covers is what it
	// wrote less the mark that was taken out of it.
	for _, hyphen := range []string{"-", "‐", "‑", "­"} {
		t.Run(hyphen, func(t *testing.T) {
			text, spans := ocr.Assemble([]ocr.Line{
				line(0, 0, 100, 20, "Viśvakoṣa"+hyphen),
				line(0, 40, 200, 60, "ṭīkā follows"),
			})
			if text != "Viśvakoṣaṭīkā follows" {
				t.Fatalf("assembled %q", text)
			}
			if len(spans) != 2 {
				t.Fatalf("wrote %d spans, want 2", len(spans))
			}
			if got := text[spans[0].Start : spans[0].Start+spans[0].Length]; got != "Viśvakoṣa" {
				t.Errorf("the first box reads %q", got)
			}
			if got := text[spans[1].Start : spans[1].Start+spans[1].Length]; got != "ṭīkā follows" {
				t.Errorf("the second box reads %q", got)
			}
		})
	}
}

func TestAPageIsCalledWhatItPrints(t *testing.T) {
	// The thirty-third page of the file, printing 2 in its corner. What the
	// page prints is what a person holding the book would say, and the number
	// is not a run of the page's prose.
	number, prose := ocr.Number([]ocr.Block{
		{Label: "number", Text: "2", Place: true},
		{Label: "text", Text: "The body begins."},
	})
	raw, _ := ocr.Write([]ocr.Page{{At: 32, Number: number, Blocks: prose}})

	text, marks := ocr.Read(raw)
	if len(marks) != 1 {
		t.Fatalf("read %d pages, want 1", len(marks))
	}
	if marks[0].Label != "2" {
		t.Errorf("the page is called %q, want %q", marks[0].Label, "2")
	}
	if got := strings.TrimSpace(text); got != "The body begins." {
		t.Errorf("the page says %q, want %q", got, "The body begins.")
	}
}

func TestAPageNothingNamesIsCalledNothing(t *testing.T) {
	// The page prints no number and the document says nothing about it. A
	// number here is one nobody could find in the book.
	raw, _ := ocr.Write([]ocr.Page{
		{At: 32, Blocks: []ocr.Block{{Label: "text", Text: "The body begins."}}},
	})

	_, marks := ocr.Read(raw)
	if len(marks) != 1 {
		t.Fatalf("read %d pages, want 1", len(marks))
	}
	if marks[0].Label != "" {
		t.Errorf("the page is called %q, and nothing says what it is called", marks[0].Label)
	}
}

func TestADocumentNamesThePagesThatPrintNoNumber(t *testing.T) {
	raw, _ := ocr.Write([]ocr.Page{
		{At: 4, Label: "v", Blocks: []ocr.Block{{Label: "text", Text: "Preface."}}},
		{At: 5, Label: "vi", Number: "6", Blocks: []ocr.Block{{Label: "text", Text: "More."}}},
	})

	_, marks := ocr.Read(raw)
	if len(marks) != 2 {
		t.Fatalf("read %d pages, want 2", len(marks))
	}
	if marks[0].Label != "v" {
		t.Errorf("the first page is called %q, want %q", marks[0].Label, "v")
	}
	// The page prints its own number and the document calls it something else.
	// The page is the one holding the book open.
	if marks[1].Label != "6" {
		t.Errorf("the second page is called %q, want %q", marks[1].Label, "6")
	}
}

func TestAPlaceRegionSayingSomethingElseNumbersNoPage(t *testing.T) {
	tests := []struct {
		says string
		want string
	}{
		{says: "2", want: "2"},
		{says: "417", want: "417"},
		{says: "ii", want: "ii"},
		{says: "XIV", want: "XIV"},
		{says: " 12 ", want: "12"},
		{says: "Chapter Two", want: ""},
		{says: "2 of 8", want: ""},
		{says: "Śrī Caitanya", want: ""},
		{says: "123456", want: ""},
	}
	for _, c := range tests {
		t.Run(c.says, func(t *testing.T) {
			number, prose := ocr.Number([]ocr.Block{
				{Label: "number", Text: c.says, Place: true},
				{Label: "text", Text: "The body."},
			})
			if number != c.want {
				t.Errorf("a page printing %q is called %q, want %q", c.says, number, c.want)
			}
			// What the region said stays out of the prose whatever it said.
			if len(prose) != 1 || prose[0].Text != "The body." {
				t.Errorf("the page says %v", prose)
			}
		})
	}
}
