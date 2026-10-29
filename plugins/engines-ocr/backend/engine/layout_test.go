package engine

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
)

// line is one run of words at a position on the page, in the form a detector
// returns it.
func line(x0, y0, x1, y1 int, text string) Line {
	return Line{Box: image.Rect(x0, y0, x1, y1), Text: text, Score: 1}
}
func TestLinesAreGroupedByWhatTheyShareVertically(t *testing.T) {
	// Two runs on one line, and a third beneath them. The one beneath is a line
	// of its own however far left it starts.
	got, _ := Assemble([]Line{
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
			got, _ := Assemble([]Line{
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
	regions := []Region{
		{Label: "text", Score: 0.9, Rect: image.Rect(0, 0, 100, 100), Order: 1},
		{Label: "text", Score: 0.8, Rect: image.Rect(2, 1, 99, 101), Order: 2},
		{Label: "paragraph_title", Score: 0.7, Rect: image.Rect(10, 10, 90, 20), Order: 3},
		{Label: "text", Score: 0.95, Rect: image.Rect(0, 200, 100, 300), Order: 4},
	}
	filtered := Distinct(regions, 0.6)

	if len(filtered) != 2 {
		var got []string
		for _, r := range filtered {
			got = append(got, r.Label)
		}
		t.Fatalf("kept %d regions (%s), want the two that do not cover each other", len(filtered), strings.Join(got, ", "))
	}
	if filtered[0].Order != 1 || filtered[1].Order != 4 {
		t.Errorf("kept the regions at %d and %d, want 1 and 4", filtered[0].Order, filtered[1].Order)
	}
}

func TestRegionsComeBackInReadingOrder(t *testing.T) {
	// The model returns regions in the order it found them, with the position of
	// each. Sorting on position keeps the second column after the first.
	filtered := Distinct([]Region{
		{Label: "text", Score: 0.9, Rect: image.Rect(200, 0, 300, 100), Order: 3},
		{Label: "text", Score: 0.9, Rect: image.Rect(0, 0, 100, 100), Order: 1},
		{Label: "text", Score: 0.9, Rect: image.Rect(0, 200, 100, 300), Order: 2},
	}, 0.6)

	for i, want := range []int{1, 2, 3} {
		if filtered[i].Order != want {
			t.Errorf("region %d is at %d, want %d", i, filtered[i].Order, want)
		}
	}
}

func TestABoxReadAsNothingIsNotAWord(t *testing.T) {
	// A detector finds a run of words and the recogniser reads nothing in it.
	// Joining that in puts two spaces where the page prints one.
	got, _ := Assemble([]Line{
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

func TestAJoinedWordPreservesTheHyphenBoxOneByteShorter(t *testing.T) {
	// The hyphen is removed from the end of the box that carried it; that box
	// covers only the first half of the word.
	text, spans := Assemble([]Line{
		line(0, 0, 100, 20, "under-"),
		line(0, 40, 200, 60, "standing follows"),
	})
	if text != "understanding follows" {
		t.Fatalf("assembled %q", text)
	}
	if len(spans) != 2 {
		t.Fatalf("wrote %d spans, want 2", len(spans))
	}
	if got := text[spans[0].Range.Start:spans[0].Range.End]; got != "under" {
		t.Errorf("the first box reads %q, want %q", got, "under")
	}
	if got := text[spans[1].Range.Start:spans[1].Range.End]; got != "standing follows" {
		t.Errorf("the second box reads %q, want %q", got, "standing follows")
	}
}

func TestAJoinedWordPreservesTheHyphenBoxShorterByTheHyphen(t *testing.T) {
	// A hyphen is one byte, or two, or three. What the box covers is what it
	// wrote less the mark that was taken out of it.
	for _, hyphen := range []string{"-", "‐", "‑", "\u00ad"} {
		t.Run(hyphen, func(t *testing.T) {
			text, spans := Assemble([]Line{
				line(0, 0, 100, 20, "Viśvakoṣa"+hyphen),
				line(0, 40, 200, 60, "ṭīkā follows"),
			})
			if text != "Viśvakoṣaṭīkā follows" {
				t.Fatalf("assembled %q", text)
			}
			if len(spans) != 2 {
				t.Fatalf("wrote %d spans, want 2", len(spans))
			}
			if got := text[spans[0].Range.Start:spans[0].Range.End]; got != "Viśvakoṣa" {
				t.Errorf("the first box reads %q", got)
			}
			if got := text[spans[1].Range.Start:spans[1].Range.End]; got != "ṭīkā follows" {
				t.Errorf("the second box reads %q", got)
			}
		})
	}
}

// createDocument is a page of rows lines, every third of which breaks a word
// with a hyphen.
func createDocument(rows int) []Line {
	lines := make([]Line, 0, rows)
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
				Assemble(lines)
			}
		})
	}
}

// createWordPage is a page of rows lines, each cut into one box per word, every
// third of which ends in a hyphen.
func createWordPage(rows int) []Line {
	words := strings.Fields("the quick brown fox jumps over the lazy dog and keeps running far")
	lines := make([]Line, 0, rows*len(words))
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
				Assemble(lines)
			}
		})
	}
}

var oldHyphen = regexp.MustCompile(`(\pL)[-‐‑\x{00ad}]$`)

// assembleByRegexp joins one-box lines that are already in reading order, by
// matching the whole text written so far.
func assembleByRegexp(lines []Line) (string, []Box) {
	var out strings.Builder
	var retained []Box
	for _, l := range lines {
		text := strings.TrimSpace(l.Text)
		if text == "" {
			continue
		}
		joined := out.String()
		if oldHyphen.MatchString(joined) {
			out.Reset()
			out.WriteString(oldHyphen.ReplaceAllString(joined, "$1"))
			if n := len(retained); n > 0 {
				retained[n-1].Range.End -= len(joined) - out.Len()
			}
		} else if joined != "" {
			out.WriteString(" ")
		}
		pos := out.Len()
		retained = append(retained, Box{Rect: l.Box, Range: ByteRange{Start: pos, End: pos + len(text)}})
		out.WriteString(text)
	}
	return out.String(), retained
}

func TestAssembleMatchesTheRegexpJoin(t *testing.T) {
	documents := map[string][]Line{"50 lines": createDocument(50), "200 lines": createDocument(200)}
	var edge []Line
	for i, text := range []string{
		"under-", "-", "-", "a-", "  ", "- x", "Viśvakoṣa\u00ad", "‐", "word -", "x‑", "1-", "-", "ṇ‐", "end",
		"bad\xff-", "\xff-", "tail-",
	} {
		edge = append(edge, line(0, i*40, 600, i*40+20, text))
	}
	documents["edge cases"] = edge

	for name, lines := range documents {
		t.Run(name, func(t *testing.T) {
			gotText, gotBoxes := Assemble(lines)
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
func createRegions(n int) []Region {
	regions := make([]Region, 0, n)
	for i := range n {
		y := (i / 3) * 100
		rect := image.Rect(0, y, 400, y+80)
		switch i % 3 {
		case 1:
			rect = rect.Add(image.Pt(5, 3))
		case 2:
			rect = image.Rect(20, y+10, 200, y+50)
		}
		regions = append(regions, Region{
			Label: "text",
			Score: float32((i*7919)%101) / 100,
			Rect:  rect,
			Order: (i * 31) % n,
		})
	}
	return regions
}

// createRows is rows lines of three skewed boxes each, in scrambled order.
func createRows(rows int) []Line {
	lines := make([]Line, 0, rows*3)
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
		scratch := make([]Region, n)
		b.Run(fmt.Sprintf("%d regions", n), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				copy(scratch, regions)
				Distinct(scratch, 0.6)
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
				Assemble(lines)
			}
		})
	}
}

func distinctByCopy(regions []Region, most float64) []Region {
	byScore := append([]Region(nil), regions...)
	sort.SliceStable(byScore, func(a, b int) bool { return byScore[a].Score > byScore[b].Score })
	filtered := make([]Region, 0, len(byScore))
	for _, r := range byScore {
		covered := false
		for _, k := range filtered {
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
			filtered = append(filtered, r)
		}
	}
	sort.SliceStable(filtered, func(a, b int) bool { return filtered[a].Order < filtered[b].Order })
	return filtered
}

func createRandomRegions(random *rand.Rand, n int) []Region {
	regions := make([]Region, n)
	for i := range regions {
		x, y := random.Intn(300), random.Intn(300)
		regions[i] = Region{
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
		got := Distinct(append([]Region(nil), regions...), 0.6)
		if !slices.Equal(got, want) {
			t.Fatalf("case %d: kept %v, want %v", i, got, want)
		}
	}
}

func groupByLists(lines []Line) [][]Line {
	sorted := append([]Line(nil), lines...)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].Box.Min.Y < sorted[b].Box.Min.Y })
	overlapOf := func(line []Line, box Line) float64 {
		top, bottom, height := box.Box.Min.Y, box.Box.Max.Y, box.Box.Dy()
		for _, l := range line {
			top, bottom, height = max(top, l.Box.Min.Y), min(bottom, l.Box.Max.Y), min(height, l.Box.Dy())
		}
		if height <= 0 {
			return 0
		}
		return float64(max(bottom-top, 0)) / float64(height)
	}
	var out [][]Line
	var current []Line
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
		lines := make([]Line, random.Intn(40))
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
		original := append([]Line(nil), lines...)
		text, boxes := Assemble(lines)
		if !slices.Equal(lines, original) {
			t.Fatalf("case %d: Assemble reordered its input", i)
		}
		// Every box's span reads back one of the words, in grouped order.
		var got []string
		for _, b := range boxes {
			got = append(got, strings.TrimSuffix(text[b.Range.Start:b.Range.End], "-")+"-")
		}
		if !slices.Equal(got, want) {
			t.Fatalf("case %d: boxes read %q, want %q", i, got, want)
		}
	}
}
