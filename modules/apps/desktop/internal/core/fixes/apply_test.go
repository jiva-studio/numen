package fixes_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/fixes"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/placed"
)

// perPage is how many printed lines a page of the little book holds.
const perPage = 3

// reading is a book read into printed lines: the prose, a mark where each page
// begins, one box for each line, and the two parts it divides into. The
// rectangle follows the line's number, so a box that keeps its place keeps its
// rectangle too.
func reading(lines []string) (string, []ocr.Mark, []placed.Box, []ocr.Part) {
	var marks []ocr.Mark
	var boxes []placed.Box
	at := 0
	for i, line := range lines {
		if i%perPage == 0 {
			marks = append(marks, ocr.Mark{Offset: at})
		}
		boxes = append(boxes, placed.Box{
			Page: i / perPage, Start: at, Length: len(line),
			MinX: 0.1, MinY: float32(i) / 100, MaxX: 0.9, MaxY: float32(i+1) / 100,
		})
		at += len(line) + len("\n")
	}
	parts := []ocr.Part{
		{Start: boxes[0].Start, Length: boxes[0].Length, Depth: 0},
		{Start: boxes[3].Start, Length: boxes[3].Length, Depth: 1},
	}
	return strings.Join(lines, "\n"), marks, boxes, parts
}

// read is the little book as the model first read it.
var read = []string{
	"Śrī Jayadeva Gosvāmī",
	"The first line ofprose",
	"and the second line.",
	"A Secodn Headng",
	"extra words to be dropped",
	"the last line.",
}

// right is the same book with the proofreader's corrections in it: one line
// lengthened, one heading lengthened, one line emptied, one line left the
// length it was.
var right = []string{
	"Śrī Jayadeva Gosvāmī",
	"The first line of prose",
	"and the second line.",
	"A Second Heading",
	"",
	"the last line!",
}

// put is the corrections as a proofreading run wrote them down.
var put = []fixes.Line{
	{At: 1, Text: right[1]},
	{At: 3, Text: right[3]},
	{At: 4, Text: right[4]},
	{At: 5, Text: right[5]},
}

func TestEveryBoxStillNamesItsWords(t *testing.T) {
	prose, marks, boxes, parts := reading(read)

	moved := fixes.Boxes(boxes, put)
	said, _, _ := fixes.Prose(prose, marks, boxes, parts, put)

	if len(moved) != len(read) {
		t.Fatalf("%d boxes, want %d", len(moved), len(read))
	}
	for i, box := range moved {
		if box.Start < 0 || box.Start+box.Length > len(said) {
			t.Fatalf("box %d covers %d..%d of %d bytes", i, box.Start, box.Start+box.Length, len(said))
		}
		if got := said[box.Start : box.Start+box.Length]; got != right[i] {
			t.Errorf("box %d says %q, want %q", i, got, right[i])
		}
	}
}

func TestACorrectedReadingIsTheReadingItShouldHaveBeen(t *testing.T) {
	prose, marks, boxes, parts := reading(read)
	wantProse, wantMarks, wantBoxes, wantParts := reading(right)

	if got := fixes.Boxes(boxes, put); !reflect.DeepEqual(got, wantBoxes) {
		t.Errorf("boxes %+v, want %+v", got, wantBoxes)
	}
	said, pages, named := fixes.Prose(prose, marks, boxes, parts, put)
	if said != wantProse {
		t.Errorf("prose %q, want %q", said, wantProse)
	}
	if !reflect.DeepEqual(pages, wantMarks) {
		t.Errorf("marks %+v, want %+v", pages, wantMarks)
	}
	if !reflect.DeepEqual(named, wantParts) {
		t.Errorf("parts %+v, want %+v", named, wantParts)
	}
}

func TestAHeadingPutRightKeepsItsOwnLength(t *testing.T) {
	prose, marks, boxes, parts := reading(read)

	// Only the second heading, which the reading had a letter short.
	_, _, named := fixes.Prose(prose, marks, boxes, parts, []fixes.Line{{At: 3, Text: right[3]}})

	if named[0].Length != parts[0].Length {
		t.Errorf("the first heading is %d bytes, want the %d it was", named[0].Length, parts[0].Length)
	}
	if named[1].Length != len(right[3]) {
		t.Errorf("the corrected heading is %d bytes, want %d", named[1].Length, len(right[3]))
	}
	if named[1].Start != parts[1].Start {
		t.Errorf("the corrected heading begins at %d, want the %d it did", named[1].Start, parts[1].Start)
	}
}

func TestALineNoBoxAnswersToIsIgnored(t *testing.T) {
	prose, marks, boxes, parts := reading(read)
	stray := []fixes.Line{{At: -1, Text: "before the book"}, {At: 99, Text: "after it"}}

	if got := fixes.Boxes(boxes, stray); !reflect.DeepEqual(got, boxes) {
		t.Errorf("boxes %+v, want them untouched", got)
	}
	said, pages, named := fixes.Prose(prose, marks, boxes, parts, stray)
	if said != prose {
		t.Errorf("prose %q, want it untouched", said)
	}
	if !reflect.DeepEqual(pages, marks) || !reflect.DeepEqual(named, parts) {
		t.Errorf("marks %+v and parts %+v, want them untouched", pages, named)
	}
}

func TestTheLastWordOnALineWins(t *testing.T) {
	prose, marks, boxes, parts := reading(read)
	twice := []fixes.Line{{At: 1, Text: "a first thought"}, {At: 1, Text: right[1]}}

	said, _, _ := fixes.Prose(prose, marks, boxes, parts, twice)
	moved := fixes.Boxes(boxes, twice)

	if got := said[moved[1].Start : moved[1].Start+moved[1].Length]; got != right[1] {
		t.Errorf("the line says %q, want %q", got, right[1])
	}
	if strings.Contains(said, "a first thought") {
		t.Errorf("prose %q still carries the correction that was replaced", said)
	}
}

func TestNoCorrectionsChangesNothing(t *testing.T) {
	prose, marks, boxes, parts := reading(read)

	for _, lines := range [][]fixes.Line{nil, {}} {
		if got := fixes.Boxes(boxes, lines); !reflect.DeepEqual(got, boxes) {
			t.Errorf("boxes %+v, want them untouched", got)
		}
		said, pages, named := fixes.Prose(prose, marks, boxes, parts, lines)
		if said != prose {
			t.Errorf("prose %q, want it untouched", said)
		}
		if !reflect.DeepEqual(pages, marks) || !reflect.DeepEqual(named, parts) {
			t.Errorf("marks %+v and parts %+v, want them untouched", pages, named)
		}
	}
}

func TestCorrectionsWrittenForOtherBytesSliceNothing(t *testing.T) {
	// A .fixes file kept beside a reading it was not made from: the boxes reach
	// past the prose, and one of them reaches backwards.
	prose, marks, boxes, parts := reading(read)
	boxes[2].Start, boxes[2].Length = len(prose)+100, 40
	boxes[4].Start = 0

	lines := []fixes.Line{{At: 2, Text: "far past the end"}, {At: 4, Text: "back at the start"}}

	said, pages, named := fixes.Prose(prose, marks, boxes, parts, lines)
	if said == "" {
		t.Errorf("prose came back empty")
	}
	moved := fixes.Boxes(boxes, lines)
	if len(moved) != len(boxes) {
		t.Errorf("%d boxes, want %d", len(moved), len(boxes))
	}
	if len(pages) != len(marks) || len(named) != len(parts) {
		t.Errorf("%d marks and %d parts, want %d and %d", len(pages), len(named), len(marks), len(parts))
	}
}
