package placed_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/placed"
)

// word is one run of prose on a page, at an offset, with a rectangle nobody
// looks at closely.
func word(page, start, length int) placed.Box {
	return placed.Box{
		Page: page, Start: start, Length: length,
		MinX: 0.1, MinY: float32(start) / 1000, MaxX: 0.9, MaxY: 0.2,
	}
}

// read is a document read into words, each five bytes with a space after.
func read(pages, each int) []placed.Box {
	var out []placed.Box
	at := 0
	for page := 0; page < pages; page++ {
		for i := 0; i < each; i++ {
			out = append(out, word(page, at, 5))
			at += 6
		}
	}
	return out
}

func TestARunIsLitWhereItWasRead(t *testing.T) {
	boxes := read(3, 4)

	// The second and third words of the first page: "6..17".
	marks := placed.Marks(boxes, 6, 12)
	if len(marks) != 1 {
		t.Fatalf("%d pages, want one", len(marks))
	}
	if marks[0].Page != 0 {
		t.Errorf("page %d", marks[0].Page)
	}
	if len(marks[0].Rects) != 2 {
		t.Errorf("%d rectangles, want the two words the run covers", len(marks[0].Rects))
	}
}

func TestARunCrossingAPageIsOnBothOfThem(t *testing.T) {
	// Four words a page, so the fourth word of page one and the first of page
	// two are one run.
	boxes := read(3, 4)

	marks := placed.Marks(boxes, 18, 12)
	if len(marks) != 2 {
		t.Fatalf("%d pages, want the two the run crosses", len(marks))
	}
	if marks[0].Page != 0 || marks[1].Page != 1 {
		t.Errorf("pages %d and %d", marks[0].Page, marks[1].Page)
	}
}

func TestAWordTheRunOnlyTouchesIsLit(t *testing.T) {
	boxes := read(1, 3)

	// One byte into the second word and one byte out of it.
	marks := placed.Marks(boxes, 7, 2)
	if len(marks) != 1 || len(marks[0].Rects) != 1 {
		t.Fatalf("%v", marks)
	}
}

func TestARunBetweenTwoWordsLightsNeither(t *testing.T) {
	// The space between the first and second word: no box holds it.
	boxes := read(1, 3)

	if marks := placed.Marks(boxes, 5, 1); marks != nil {
		t.Errorf("the gap lit %v", marks)
	}
}

func TestNothingIsAskedForAndNothingIsLit(t *testing.T) {
	boxes := read(2, 2)

	if marks := placed.Marks(boxes, 0, 0); marks != nil {
		t.Errorf("a run of nothing lit %v", marks)
	}
	if marks := placed.Marks(nil, 0, 10); marks != nil {
		t.Errorf("a document nobody placed lit %v", marks)
	}
	if marks := placed.Marks(boxes, 9000, 10); marks != nil {
		t.Errorf("a run past the end lit %v", marks)
	}
}

func TestThePagesComeBackInTheOrderTheyAreRead(t *testing.T) {
	boxes := read(4, 2)

	marks := placed.Marks(boxes, 0, 48)
	var pages []int
	for _, one := range marks {
		pages = append(pages, one.Page)
	}
	if want := []int{0, 1, 2, 3}; !reflect.DeepEqual(pages, want) {
		t.Errorf("pages %v, want %v", pages, want)
	}
}
