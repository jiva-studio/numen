package domain_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// perPage is how many printed lines a page of the little book holds.
const perPage = 3

// createRecognition is a book recognised into printed lines: the prose, a mark where each page
// begins, one box for each line, and the two parts it divides into. The
// rectangle follows the line's number, so a box that keeps its place keeps its
// rectangle too.
func createRecognition(lines []string) (string, []domain.OCRPageStart, []domain.HighlightBox, []domain.OCRPart) {
	var marks []domain.OCRPageStart
	var boxes []domain.HighlightBox
	offset := 0
	for i, line := range lines {
		if i%perPage == 0 {
			marks = append(marks, domain.OCRPageStart{Offset: offset})
		}
		boxes = append(boxes, domain.HighlightBox{
			Page:     i / perPage,
			ByteSpan: domain.ByteSpan{From: offset, To: offset + len(line)},
			Rect: domain.Rect{
				MinX: 0.1, MinY: float32(i) / 100, MaxX: 0.9, MaxY: float32(i+1) / 100,
			},
		})
		offset += len(line) + len("\n")
	}
	parts := []domain.OCRPart{
		{Start: boxes[0].From, Length: boxes[0].Len(), Depth: 0},
		{Start: boxes[3].From, Length: boxes[3].Len(), Depth: 1},
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
var put = []domain.CorrectionLine{
	{Number: 1, Text: right[1]},
	{Number: 3, Text: right[3]},
	{Number: 4, Text: right[4]},
	{Number: 5, Text: right[5]},
}

func TestEveryBoxStillNamesItsWords(t *testing.T) {
	prose, marks, boxes, parts := createRecognition(read)

	moved := domain.GetCorrectedBoxes(boxes, put)
	response, _, _ := domain.GetCorrectedProse(prose, marks, boxes, parts, put)

	if len(moved) != len(read) {
		t.Fatalf("%d boxes, want %d", len(moved), len(read))
	}
	for i, box := range moved {
		if box.From < 0 || box.To > len(response) {
			t.Fatalf("box %d covers %d..%d of %d bytes", i, box.From, box.To, len(response))
		}
		if got := response[box.From:box.To]; got != right[i] {
			t.Errorf("box %d says %q, want %q", i, got, right[i])
		}
	}
}

func TestACorrectedRecognitionIsTheRecognitionItShouldHaveBeen(t *testing.T) {
	prose, marks, boxes, parts := createRecognition(read)
	wantProse, wantMarks, wantBoxes, wantParts := createRecognition(right)

	if got := domain.GetCorrectedBoxes(boxes, put); !reflect.DeepEqual(got, wantBoxes) {
		t.Errorf("boxes %+v, want %+v", got, wantBoxes)
	}
	response, pages, named := domain.GetCorrectedProse(prose, marks, boxes, parts, put)
	if response != wantProse {
		t.Errorf("prose %q, want %q", response, wantProse)
	}
	if !reflect.DeepEqual(pages, wantMarks) {
		t.Errorf("marks %+v, want %+v", pages, wantMarks)
	}
	if !reflect.DeepEqual(named, wantParts) {
		t.Errorf("parts %+v, want %+v", named, wantParts)
	}
}

func TestAHeadingInsertRightPreservesItsOwnLength(t *testing.T) {
	prose, marks, boxes, parts := createRecognition(read)

	// Only the second heading, which the recognition had a letter short.
	_, _, named := domain.GetCorrectedProse(prose, marks, boxes, parts, []domain.CorrectionLine{{Number: 3, Text: right[3]}})

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
	prose, marks, boxes, parts := createRecognition(read)
	stray := []domain.CorrectionLine{{Number: -1, Text: "before the book"}, {Number: 99, Text: "after it"}}

	if got := domain.GetCorrectedBoxes(boxes, stray); !reflect.DeepEqual(got, boxes) {
		t.Errorf("boxes %+v, want them untouched", got)
	}
	response, pages, named := domain.GetCorrectedProse(prose, marks, boxes, parts, stray)
	if response != prose {
		t.Errorf("prose %q, want it untouched", response)
	}
	if !reflect.DeepEqual(pages, marks) || !reflect.DeepEqual(named, parts) {
		t.Errorf("marks %+v and parts %+v, want them untouched", pages, named)
	}
}

func TestTheLastWordOnALineWins(t *testing.T) {
	prose, marks, boxes, parts := createRecognition(read)
	twice := []domain.CorrectionLine{{Number: 1, Text: "a first thought"}, {Number: 1, Text: right[1]}}

	response, _, _ := domain.GetCorrectedProse(prose, marks, boxes, parts, twice)
	moved := domain.GetCorrectedBoxes(boxes, twice)

	if got := response[moved[1].From:moved[1].To]; got != right[1] {
		t.Errorf("the line says %q, want %q", got, right[1])
	}
	if strings.Contains(response, "a first thought") {
		t.Errorf("prose %q still carries the correction that was replaced", response)
	}
}

func TestNoCorrectionsChangesNothing(t *testing.T) {
	prose, marks, boxes, parts := createRecognition(read)

	for _, lines := range [][]domain.CorrectionLine{nil, {}} {
		if got := domain.GetCorrectedBoxes(boxes, lines); !reflect.DeepEqual(got, boxes) {
			t.Errorf("boxes %+v, want them untouched", got)
		}
		response, pages, named := domain.GetCorrectedProse(prose, marks, boxes, parts, lines)
		if response != prose {
			t.Errorf("prose %q, want it untouched", response)
		}
		if !reflect.DeepEqual(pages, marks) || !reflect.DeepEqual(named, parts) {
			t.Errorf("marks %+v and parts %+v, want them untouched", pages, named)
		}
	}
}

func TestCorrectionsSavedForOtherBytesSliceNothing(t *testing.T) {
	// A .corrected file kept beside a recognition it was not made from: the boxes reach
	// past the prose, and one of them reaches backwards.
	prose, marks, boxes, parts := createRecognition(read)
	boxes[2].ByteSpan = domain.ByteSpan{From: len(prose) + 100, To: len(prose) + 140}
	boxes[4].From = 0

	lines := []domain.CorrectionLine{{Number: 2, Text: "far past the end"}, {Number: 4, Text: "back at the start"}}

	response, pages, named := domain.GetCorrectedProse(prose, marks, boxes, parts, lines)
	if response == "" {
		t.Errorf("prose came back empty")
	}
	moved := domain.GetCorrectedBoxes(boxes, lines)
	if len(moved) != len(boxes) {
		t.Errorf("%d boxes, want %d", len(moved), len(boxes))
	}
	if len(pages) != len(marks) || len(named) != len(parts) {
		t.Errorf("%d marks and %d parts, want %d and %d", len(pages), len(named), len(marks), len(parts))
	}
}

func TestLinesNamedOutOfOrderAndTwiceReadAsIfInOrder(t *testing.T) {
	prose, marks, boxes, parts := createRecognition(read)
	shuffled := []domain.CorrectionLine{put[3], {Number: 1, Text: "first"}, put[1], put[0], put[2], {Number: 3, Text: "A Second Heading"}}
	ordered := []domain.CorrectionLine{put[0], {Number: 3, Text: "A Second Heading"}, put[2], put[3]}
	ordered[0] = domain.CorrectionLine{Number: 1, Text: right[1]}

	gotProse, gotMarks, gotParts := domain.GetCorrectedProse(prose, marks, boxes, parts, shuffled)
	wantProse, wantMarks, wantParts := domain.GetCorrectedProse(prose, marks, boxes, parts, ordered)
	if gotProse != wantProse || !reflect.DeepEqual(gotMarks, wantMarks) || !reflect.DeepEqual(gotParts, wantParts) {
		t.Errorf("shuffled lines read differently from ordered ones")
	}
	if got, want := domain.GetCorrectedBoxes(boxes, shuffled), domain.GetCorrectedBoxes(boxes, ordered); !reflect.DeepEqual(got, want) {
		t.Errorf("Boxes = %v, want %v", got, want)
	}
}
