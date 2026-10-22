package ocrartifact_test

import (
	"image"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/ocrartifact"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestAnArtifactReturnsWhatEachPageReturns(t *testing.T) {
	pages := []ocrartifact.Page{
		{Blocks: []domain.OCRBlock{{Label: "text", Text: "Preface."}}},
		{Blocks: []domain.OCRBlock{
			{Label: "paragraph_title", Text: "THE FIRST PART"},
			{Label: "text", Text: "The body begins."},
		}},
	}
	raw, _, _ := ocrartifact.Write(pages)

	text, marks := ocrartifact.Read(raw)
	if len(marks) != 2 {
		t.Fatalf("read %d pages, want 2", len(marks))
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
	text, marks := ocrartifact.Read([]byte("just words, and no pages named"))

	if len(marks) != 0 {
		t.Errorf("found %d pages in a file that names none", len(marks))
	}
	if text != "just words, and no pages named" {
		t.Errorf("read %q", text)
	}
}

func TestAPageWithNothingOnItIsStillAPage(t *testing.T) {
	raw, _, _ := ocrartifact.Write([]ocrartifact.Page{
		{Blocks: []domain.OCRBlock{{Label: "text", Text: "Something."}}},
		{},
		{Blocks: []domain.OCRBlock{{Label: "text", Text: "Something else."}}},
	})

	_, marks := ocrartifact.Read(raw)
	if len(marks) != 3 {
		t.Fatalf("read %d pages, want 3 — a blank page is a fact about the document", len(marks))
	}
}

func TestWritingAndReadingAgreeAboutEveryOffset(t *testing.T) {
	pages := []ocrartifact.Page{
		{Blocks: []domain.OCRBlock{{Label: "text", Text: "Alpha."}}},
		{Blocks: []domain.OCRBlock{{Label: "text", Text: "Beta."}, {Label: "text", Text: "Gamma."}}},
		{Blocks: []domain.OCRBlock{{Label: "text", Text: "Delta."}}},
	}
	raw, _, _ := ocrartifact.Write(pages)
	text, marks := ocrartifact.Read(raw)

	// Every mark is inside the text and they ascend, which is what Locate
	// searches through.
	for i, mark := range marks {
		if mark.Offset < 0 || mark.Offset > len(text) {
			t.Errorf("page %d begins at %d, and the text is %d long", i, mark.Offset, len(text))
		}
		if i > 0 && mark.Offset < marks[i-1].Offset {
			t.Errorf("page %d begins before the page before it", i)
		}
	}
	// The first page's prose is where it says it is.
	if got := strings.TrimSpace(text[marks[0].Offset:marks[1].Offset]); got != "Alpha." {
		t.Errorf("the first page says %q, want %q", got, "Alpha.")
	}
}

func TestALineSavedForTheWriterIsNotProse(t *testing.T) {
	// A run stopped part way writes down how far it got. That line is not what
	// the page says, and an offset into the prose must not count it.
	first, _, _ := ocrartifact.Write([]ocrartifact.Page{
		{Blocks: []domain.OCRBlock{{Label: "text", Text: "Alpha."}}},
	})
	second, _, _ := ocrartifact.Write([]ocrartifact.Page{
		{Blocks: []domain.OCRBlock{{Label: "text", Text: "Beta."}}},
	})
	raw := append([]byte("\x00pages 2\n"), first...)
	raw = append(raw, []byte("\x00pages 4\n")...)
	raw = append(raw, second...)

	text, marks := ocrartifact.Read(raw)
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

func TestEveryBoxReturnsWhereItsWordsAreInTheProse(t *testing.T) {
	// Two pages, and the first of them two regions. The offsets are in the
	// prose, so the second page's boxes are past everything the first says.
	first, firstBoxes := "Alpha beta gamma", []domain.OCRBox{
		{Rect: image.Rect(0, 0, 50, 20), Span: domain.ByteSpan{From: 0, To: 5}},
		{Rect: image.Rect(60, 0, 100, 20), Span: domain.ByteSpan{From: 6, To: 10}},
		{Rect: image.Rect(0, 40, 60, 60), Span: domain.ByteSpan{From: 11, To: 16}},
	}
	last, lastBoxes := "Epsilon zeta", []domain.OCRBox{
		{Rect: image.Rect(0, 0, 70, 20), Span: domain.ByteSpan{From: 0, To: 7}},
		{Rect: image.Rect(80, 0, 120, 20), Span: domain.ByteSpan{From: 8, To: 12}},
	}
	opening := []domain.OCRBlock{
		{Label: "text", Text: first, Boxes: firstBoxes},
		// A region read by something that reports no rectangles. It says
		// what it says and the prose after it moves along by that much.
		{Label: "text", Text: "Delta."},
	}
	pages := []ocrartifact.Page{
		{Index: 0, Size: image.Pt(600, 800), Blocks: opening},
		{Index: 1, Size: image.Pt(600, 800), Blocks: []domain.OCRBlock{
			{Label: "text", Text: last, Boxes: lastBoxes},
		}},
	}
	raw, boxes, _ := ocrartifact.Write(pages)
	text, _ := ocrartifact.Read(raw)

	want := []string{"Alpha", "beta", "gamma", "Epsilon", "zeta"}
	if len(boxes) != len(want) {
		t.Fatalf("wrote %d boxes, want %d", len(boxes), len(want))
	}
	for i, box := range boxes {
		if box.From < 0 || box.To > len(text) {
			t.Fatalf("box %d covers %d..%d, and the prose is %d long", i, box.From, box.To, len(text))
		}
		if got := text[box.From:box.To]; got != want[i] {
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
		for _, item := range []float32{box.MinX, box.MinY, box.MaxX, box.MaxY} {
			if item < 0 || item > 1 {
				t.Errorf("box %d reaches %v, and the page is one wide and one high", i, item)
			}
		}
	}
}

func TestAPageNothingWasMeasuredOnHasNoBoxes(t *testing.T) {
	// A page with no size gives no fraction of itself to divide a rectangle by.
	raw, boxes, _ := ocrartifact.Write([]ocrartifact.Page{
		{Index: 0, Blocks: []domain.OCRBlock{{
			Label: "text",
			Text:  "Alpha beta",
			Boxes: []domain.OCRBox{{Rect: image.Rect(0, 0, 50, 20), Span: domain.ByteSpan{From: 0, To: 5}}},
		}}},
	})

	if len(boxes) != 0 {
		t.Errorf("wrote %d boxes for a page nothing was measured on", len(boxes))
	}
	text, marks := ocrartifact.Read(raw)
	if len(marks) != 1 || !strings.Contains(text, "Alpha beta") {
		t.Errorf("the page says %q", text)
	}
}

// A heading is where a part of the document begins, and a part carries the run
// of prose the heading is, not the words themselves. A scan that read the
// heading badly still says where its part starts.
func TestAHeadingReturnsWhereAPartOfTheDocumentBegins(t *testing.T) {
	raw, _, parts := ocrartifact.Write([]ocrartifact.Page{
		{Index: 0, Blocks: []domain.OCRBlock{
			{Label: "doc_title", Text: "IAYADEVA GOSVAMI", IsHeading: true, Depth: 0},
			{Label: "text", Text: "He was born in Kenduli."},
			{Label: "paragraph_title", Text: "His Youth", IsHeading: true, Depth: 1},
		}},
		{Index: 1, Blocks: []domain.OCRBlock{
			{Label: "text", Text: "The village stands there still."},
			{Label: "paragraph_title", Text: "The Journey", IsHeading: true, Depth: 1},
		}},
	})
	text, _ := ocrartifact.Read(raw)

	want := []struct {
		title string
		depth int
	}{
		{"IAYADEVA GOSVAMI", 0},
		{"His Youth", 1},
		{"The Journey", 1},
	}
	if len(parts) != len(want) {
		t.Fatalf("wrote %d parts, want %d", len(parts), len(want))
	}
	pos := -1
	for i, part := range parts {
		if part.Start <= pos {
			t.Errorf("part %d begins at %d, and the one before it at %d", i, part.Start, pos)
		}
		pos = part.Start
		if part.Start < 0 || part.Start+part.Length > len(text) {
			t.Fatalf("part %d covers %d..%d, and the prose is %d long",
				i, part.Start, part.Start+part.Length, len(text))
		}
		if got := text[part.Start : part.Start+part.Length]; got != want[i].title {
			t.Errorf("part %d reads %q, want %q", i, got, want[i].title)
		}
		if part.Depth != want[i].depth {
			t.Errorf("%q sits at depth %d, want %d", want[i].title, part.Depth, want[i].depth)
		}
	}
}
