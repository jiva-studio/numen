package highlight_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// word is one run of prose on a page, at an offset, with a rectangle nobody
// looks at closely.
func word(page, start, length int) domain.HighlightBox {
	return domain.HighlightBox{
		Page:     page,
		ByteSpan: domain.ByteSpan{From: start, To: start + length},
		Rect:     domain.Rect{MinX: 0.1, MinY: float32(start) / 1000, MaxX: 0.9, MaxY: 0.2},
	}
}

// run is a span of the prose, by where it begins and how far it reaches.
func run(start, length int) domain.ByteSpan {
	return domain.ByteSpan{From: start, To: start + length}
}

// read is a document read into words, each five bytes with a space after.
func read(pages, each int) []domain.HighlightBox {
	var out []domain.HighlightBox
	pos := 0
	for page := 0; page < pages; page++ {
		for i := 0; i < each; i++ {
			out = append(out, word(page, pos, 5))
			pos += 6
		}
	}
	return out
}

func TestARunIsLitWhereItWasRead(t *testing.T) {
	boxes := read(3, 4)

	// The second and third words of the first page: "6..17".
	matchedBoxes := highlight.GetBoxesOver(boxes, run(6, 12))
	if len(matchedBoxes) != 2 {
		t.Fatalf("%d boxes, want the two words the run covers", len(matchedBoxes))
	}
	if matchedBoxes[0].Page != 0 || matchedBoxes[1].Page != 0 {
		t.Errorf("pages %d and %d", matchedBoxes[0].Page, matchedBoxes[1].Page)
	}
}

func TestARunCrossingAPageIsOnBothOfThem(t *testing.T) {
	// Four words a page, so the fourth word of page one and the first of page
	// two are one run.
	boxes := read(3, 4)

	matchedBoxes := highlight.GetBoxesOver(boxes, run(18, 12))
	if len(matchedBoxes) != 2 {
		t.Fatalf("%d boxes, want the two words the run crosses", len(matchedBoxes))
	}
	if matchedBoxes[0].Page != 0 || matchedBoxes[1].Page != 1 {
		t.Errorf("pages %d and %d", matchedBoxes[0].Page, matchedBoxes[1].Page)
	}
}

func TestAWordTheRunOnlyTouchesIsLit(t *testing.T) {
	boxes := read(1, 3)

	// One byte into the second word and one byte out of it.
	matchedBoxes := highlight.GetBoxesOver(boxes, run(7, 2))
	if len(matchedBoxes) != 1 {
		t.Fatalf("%v", matchedBoxes)
	}
}

func TestARunBetweenTwoWordsLightsNeither(t *testing.T) {
	// The space between the first and second word: no box holds it.
	boxes := read(1, 3)

	if matchedBoxes := highlight.GetBoxesOver(boxes, run(5, 1)); matchedBoxes != nil {
		t.Errorf("the gap lit %v", matchedBoxes)
	}
}

func TestNothingIsAskedForAndNothingIsLit(t *testing.T) {
	boxes := read(2, 2)

	if matchedBoxes := highlight.GetBoxesOver(boxes, run(0, 0)); matchedBoxes != nil {
		t.Errorf("a run of nothing lit %v", matchedBoxes)
	}
	if matchedBoxes := highlight.GetBoxesOver(nil, run(0, 10)); matchedBoxes != nil {
		t.Errorf("a document nobody lit lit %v", matchedBoxes)
	}
	if matchedBoxes := highlight.GetBoxesOver(boxes, run(9000, 10)); matchedBoxes != nil {
		t.Errorf("a run past the end lit %v", matchedBoxes)
	}
}

func TestTheBoxesComeBackInTheOrderTheyAreRead(t *testing.T) {
	boxes := read(4, 2)

	var offset []int
	for _, one := range highlight.GetBoxesOver(boxes, run(0, 48)) {
		offset = append(offset, one.Page)
	}
	if want := []int{0, 0, 1, 1, 2, 2, 3, 3}; !reflect.DeepEqual(offset, want) {
		t.Errorf("pages %v, want %v", offset, want)
	}
}
