package ocr_test

import (
	"fmt"
	"image"
	"math/rand"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
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
		{name: "a soft hyphen joins", ends: "under\u00ad", next: "standing follows", want: "understanding follows"},
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
		{Blocks: []ocr.Block{{Label: "text", Text: "Preface."}}},
		{Blocks: []ocr.Block{
			{Label: "paragraph_title", Text: "THE FIRST PART"},
			{Label: "text", Text: "The body begins."},
		}},
	}
	raw, _, _ := ocr.Write(pages)

	text, marks := ocr.Read(raw)
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
	text, marks := ocr.Read([]byte("just words, and no pages named"))

	if len(marks) != 0 {
		t.Errorf("found %d pages in a file that names none", len(marks))
	}
	if text != "just words, and no pages named" {
		t.Errorf("read %q", text)
	}
}

func TestAPageWithNothingOnItIsStillAPage(t *testing.T) {
	raw, _, _ := ocr.Write([]ocr.Page{
		{Blocks: []ocr.Block{{Label: "text", Text: "Something."}}},
		{},
		{Blocks: []ocr.Block{{Label: "text", Text: "Something else."}}},
	})

	_, marks := ocr.Read(raw)
	if len(marks) != 3 {
		t.Fatalf("read %d pages, want 3 — a blank page is a fact about the document", len(marks))
	}
}

func TestWritingAndReadingAgreeAboutEveryOffset(t *testing.T) {
	pages := []ocr.Page{
		{Blocks: []ocr.Block{{Label: "text", Text: "Alpha."}}},
		{Blocks: []ocr.Block{{Label: "text", Text: "Beta."}, {Label: "text", Text: "Gamma."}}},
		{Blocks: []ocr.Block{{Label: "text", Text: "Delta."}}},
	}
	raw, _, _ := ocr.Write(pages)
	text, marks := ocr.Read(raw)

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
	first, _, _ := ocr.Write([]ocr.Page{
		{Blocks: []ocr.Block{{Label: "text", Text: "Alpha."}}},
	})
	second, _, _ := ocr.Write([]ocr.Page{
		{Blocks: []ocr.Block{{Label: "text", Text: "Beta."}}},
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
	first, firstBoxes := ocr.Assemble([]ocr.Line{
		line(0, 0, 50, 20, "Alpha"),
		line(60, 0, 100, 20, "beta"),
		line(0, 40, 60, 60, "gamma"),
	})
	last, lastBoxes := ocr.Assemble([]ocr.Line{
		line(0, 0, 70, 20, "Epsilon"),
		line(80, 0, 120, 20, "zeta"),
	})
	opening := []ocr.Block{
		{Label: "text", Text: first, Boxes: firstBoxes},
		// A region read by something that reports no rectangles. It says
		// what it says and the prose after it moves along by that much.
		{Label: "text", Text: "Delta."},
	}
	pages := []ocr.Page{
		{Index: 0, Size: image.Pt(600, 800), Blocks: opening},
		{Index: 1, Size: image.Pt(600, 800), Blocks: []ocr.Block{
			{Label: "text", Text: last, Boxes: lastBoxes},
		}},
	}
	raw, boxes, _ := ocr.Write(pages)
	text, _ := ocr.Read(raw)

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
	if got := text[spans[0].Span.From:spans[0].Span.To]; got != "under" {
		t.Errorf("the first box reads %q, want %q", got, "under")
	}
	if got := text[spans[1].Span.From:spans[1].Span.To]; got != "standing follows" {
		t.Errorf("the second box reads %q, want %q", got, "standing follows")
	}
}

func TestAPageNothingWasMeasuredOnHasNoBoxes(t *testing.T) {
	// A page with no size gives no fraction of itself to divide a rectangle by.
	raw, boxes, _ := ocr.Write([]ocr.Page{
		{Index: 0, Blocks: []ocr.Block{{
			Label: "text",
			Text:  "Alpha beta",
			Boxes: []ocr.Box{{Rect: image.Rect(0, 0, 50, 20), Span: domain.ByteSpan{From: 0, To: 5}}},
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
	for _, hyphen := range []string{"-", "‐", "‑", "\u00ad"} {
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
			if got := text[spans[0].Span.From:spans[0].Span.To]; got != "Viśvakoṣa" {
				t.Errorf("the first box reads %q", got)
			}
			if got := text[spans[1].Span.From:spans[1].Span.To]; got != "ṭīkā follows" {
				t.Errorf("the second box reads %q", got)
			}
		})
	}
}

// A heading is where a part of the document begins, and a part carries the run
// of prose the heading is, not the words themselves. A scan that read the
// heading badly still says where its part starts.
func TestAHeadingSaysWhereAPartOfTheDocumentBegins(t *testing.T) {
	raw, _, parts := ocr.Write([]ocr.Page{
		{Index: 0, Blocks: []ocr.Block{
			{Label: "doc_title", Text: "IAYADEVA GOSVAMI", IsHeading: true, Depth: 0},
			{Label: "text", Text: "He was born in Kenduli."},
			{Label: "paragraph_title", Text: "His Youth", IsHeading: true, Depth: 1},
		}},
		{Index: 1, Blocks: []ocr.Block{
			{Label: "text", Text: "The village stands there still."},
			{Label: "paragraph_title", Text: "The Journey", IsHeading: true, Depth: 1},
		}},
	})
	text, _ := ocr.Read(raw)

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
	at := -1
	for i, part := range parts {
		if part.Start <= at {
			t.Errorf("part %d begins at %d, and the one before it at %d", i, part.Start, at)
		}
		at = part.Start
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

// createDocument is a page of rows lines, every third of which breaks a word
// with a hyphen.
func createDocument(rows int) []ocr.Line {
	lines := make([]ocr.Line, 0, rows)
	for i := range rows {
		text := "the quick brown fox jumps over the lazy dog and keeps running far"
		switch i % 3 {
		case 0:
			text += " under-"
		case 1:
			text = "standing " + text
		}
		lines = append(lines, line(0, i*40, 600, i*40+20, text))
	}
	return lines
}

func BenchmarkAssemble(b *testing.B) {
	for _, rows := range []int{50, 200} {
		lines := createDocument(rows)
		b.Run(fmt.Sprintf("%d lines", rows), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				ocr.Assemble(lines)
			}
		})
	}
}

// createWordPage is a page of rows lines, each cut into one box per word, every
// third of which ends in a hyphen.
func createWordPage(rows int) []ocr.Line {
	words := strings.Fields("the quick brown fox jumps over the lazy dog and keeps running far")
	lines := make([]ocr.Line, 0, rows*len(words))
	for i := range rows {
		for j, word := range words {
			if i%3 == 0 && j == len(words)-1 {
				word += "-"
			}
			lines = append(lines, line(j*50, i*40, j*50+45, i*40+20, word))
		}
	}
	return lines
}

func BenchmarkAssembleWords(b *testing.B) {
	for _, rows := range []int{40, 120} {
		lines := createWordPage(rows)
		b.Run(fmt.Sprintf("%d lines", rows), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				ocr.Assemble(lines)
			}
		})
	}
}

var oldHyphen = regexp.MustCompile(`(\pL)[-‐‑\x{00ad}]$`)

// assembleByRegexp joins one-box lines that are already in reading order, by
// matching the whole text written so far.
func assembleByRegexp(lines []ocr.Line) (string, []ocr.Box) {
	var out strings.Builder
	var kept []ocr.Box
	for _, l := range lines {
		text := strings.TrimSpace(l.Text)
		if text == "" {
			continue
		}
		joined := out.String()
		if oldHyphen.MatchString(joined) {
			out.Reset()
			out.WriteString(oldHyphen.ReplaceAllString(joined, "$1"))
			if n := len(kept); n > 0 {
				kept[n-1].Span.To -= len(joined) - out.Len()
			}
		} else if joined != "" {
			out.WriteString(" ")
		}
		at := out.Len()
		kept = append(kept, ocr.Box{Rect: l.Box, Span: domain.ByteSpan{From: at, To: at + len(text)}})
		out.WriteString(text)
	}
	return out.String(), kept
}

func TestAssembleMatchesTheRegexpJoin(t *testing.T) {
	documents := map[string][]ocr.Line{"50 lines": createDocument(50), "200 lines": createDocument(200)}
	var edge []ocr.Line
	for i, text := range []string{
		"under-", "-", "-", "a-", "  ", "- x", "Viśvakoṣa\u00ad", "‐", "word -", "x‑", "1-", "-", "ṇ‐", "end",
		"bad\xff-", "\xff-", "tail-",
	} {
		edge = append(edge, line(0, i*40, 600, i*40+20, text))
	}
	documents["edge cases"] = edge

	for name, lines := range documents {
		t.Run(name, func(t *testing.T) {
			gotText, gotBoxes := ocr.Assemble(lines)
			wantText, wantBoxes := assembleByRegexp(lines)
			if gotText != wantText {
				t.Errorf("assembled %q, want %q", gotText, wantText)
			}
			if !reflect.DeepEqual(gotBoxes, wantBoxes) {
				t.Errorf("boxes %v, want %v", gotBoxes, wantBoxes)
			}
		})
	}
}

// createRegions is n regions in a scrambled reading order, a third of which
// repeat another and a third of which sit inside another.
func createRegions(n int) []ocr.Region {
	regions := make([]ocr.Region, 0, n)
	for i := range n {
		y := (i / 3) * 100
		rect := image.Rect(0, y, 400, y+80)
		switch i % 3 {
		case 1:
			rect = rect.Add(image.Pt(5, 3))
		case 2:
			rect = image.Rect(20, y+10, 200, y+50)
		}
		regions = append(regions, ocr.Region{
			Label: "text",
			Score: float32((i*7919)%101) / 100,
			Rect:  rect,
			Order: (i * 31) % n,
		})
	}
	return regions
}

// createRows is rows lines of three skewed boxes each, in scrambled order.
func createRows(rows int) []ocr.Line {
	lines := make([]ocr.Line, 0, rows*3)
	for i := range rows * 3 {
		row, col := (i*7)%rows, (i*5)%3
		skew := (i % 4) - 1
		lines = append(lines, line(col*200, row*40+skew, col*200+190, row*40+20+skew, "word"))
	}
	return lines
}

func BenchmarkDistinct(b *testing.B) {
	for _, n := range []int{20, 100, 400} {
		regions := createRegions(n)
		scratch := make([]ocr.Region, n)
		b.Run(fmt.Sprintf("%d regions", n), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				copy(scratch, regions)
				ocr.Distinct(scratch, 0.6)
			}
		})
	}
}

func BenchmarkGroup(b *testing.B) {
	for _, rows := range []int{50, 200} {
		lines := createRows(rows)
		b.Run(fmt.Sprintf("%d rows", rows), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				ocr.Assemble(lines)
			}
		})
	}
}

func distinctByCopy(regions []ocr.Region, most float64) []ocr.Region {
	byScore := append([]ocr.Region(nil), regions...)
	sort.SliceStable(byScore, func(a, b int) bool { return byScore[a].Score > byScore[b].Score })
	kept := make([]ocr.Region, 0, len(byScore))
	for _, r := range byScore {
		covered := false
		for _, k := range kept {
			both := r.Rect.Intersect(k.Rect)
			if both.Empty() {
				continue
			}
			common := float64(both.Dx() * both.Dy())
			either := float64(r.Rect.Dx()*r.Rect.Dy()) + float64(k.Rect.Dx()*k.Rect.Dy()) - common
			if (either > 0 && common/either > most) || common > 0.9*float64(r.Rect.Dx()*r.Rect.Dy()) {
				covered = true
				break
			}
		}
		if !covered {
			kept = append(kept, r)
		}
	}
	sort.SliceStable(kept, func(a, b int) bool { return kept[a].Order < kept[b].Order })
	return kept
}

func createRandomRegions(random *rand.Rand, n int) []ocr.Region {
	regions := make([]ocr.Region, n)
	for i := range regions {
		x, y := random.Intn(300), random.Intn(300)
		regions[i] = ocr.Region{
			Label: fmt.Sprint(i),
			Score: float32(random.Intn(5)) / 4,
			Rect:  image.Rect(x, y, x+random.Intn(200), y+random.Intn(200)),
			Order: random.Intn(6),
		}
	}
	return regions
}

func TestDistinctMatchesTheCopyingFilter(t *testing.T) {
	random := rand.New(rand.NewSource(364))
	for i := range 300 {
		regions := createRandomRegions(random, random.Intn(60))
		want := distinctByCopy(regions, 0.6)
		got := ocr.Distinct(append([]ocr.Region(nil), regions...), 0.6)
		if !slices.Equal(got, want) {
			t.Fatalf("case %d: kept %v, want %v", i, got, want)
		}
	}
}

func groupByLists(lines []ocr.Line) [][]ocr.Line {
	sorted := append([]ocr.Line(nil), lines...)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].Box.Min.Y < sorted[b].Box.Min.Y })
	overlapOf := func(line []ocr.Line, box ocr.Line) float64 {
		top, bottom, height := box.Box.Min.Y, box.Box.Max.Y, box.Box.Dy()
		for _, l := range line {
			top, bottom, height = max(top, l.Box.Min.Y), min(bottom, l.Box.Max.Y), min(height, l.Box.Dy())
		}
		if height <= 0 {
			return 0
		}
		return float64(max(bottom-top, 0)) / float64(height)
	}
	var out [][]ocr.Line
	var current []ocr.Line
	flush := func() {
		sort.SliceStable(current, func(a, b int) bool { return current[a].Box.Min.X < current[b].Box.Min.X })
		out = append(out, current)
	}
	for _, l := range sorted {
		if len(current) > 0 && overlapOf(current, l) < 0.5 {
			flush()
			current = nil
		}
		current = append(current, l)
	}
	if len(current) > 0 {
		flush()
	}
	return out
}

func TestAssembleMatchesTheListGrouping(t *testing.T) {
	random := rand.New(rand.NewSource(364))
	for i := range 300 {
		lines := make([]ocr.Line, random.Intn(40))
		for j := range lines {
			x, y := random.Intn(400), random.Intn(120)
			lines[j] = line(x, y, x+random.Intn(100), y+random.Intn(30), fmt.Sprintf("w%d-", random.Intn(9)))
		}
		var want []string
		for _, l := range groupByLists(lines) {
			for _, one := range l {
				want = append(want, one.Text)
			}
		}
		original := append([]ocr.Line(nil), lines...)
		text, boxes := ocr.Assemble(lines)
		if !slices.Equal(lines, original) {
			t.Fatalf("case %d: Assemble reordered its input", i)
		}
		// Every box's span reads back one of the words, in grouped order.
		var got []string
		for _, b := range boxes {
			got = append(got, strings.TrimSuffix(text[b.Span.From:b.Span.To], "-")+"-")
		}
		if !slices.Equal(got, want) {
			t.Fatalf("case %d: boxes read %q, want %q", i, got, want)
		}
	}
}
