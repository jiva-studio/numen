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
	got := ocr.Assemble([]ocr.Line{
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
			got := ocr.Assemble([]ocr.Line{
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
	raw := ocr.Write(pages)

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
	raw := ocr.Write([]ocr.Page{
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
	text, marks := ocr.Read(ocr.Write(pages))

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
	got := ocr.Assemble([]ocr.Line{
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
	raw := append([]byte("\x00pages 2\n"), ocr.Write([]ocr.Page{
		{Label: "1", Blocks: []ocr.Block{{Label: "text", Text: "Alpha."}}},
	})...)
	raw = append(raw, []byte("\x00pages 4\n")...)
	raw = append(raw, ocr.Write([]ocr.Page{
		{Label: "2", Blocks: []ocr.Block{{Label: "text", Text: "Beta."}}},
	})...)

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
