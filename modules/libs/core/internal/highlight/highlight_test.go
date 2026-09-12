package highlight_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
)

// word is one run of prose on a page, at an offset, with a rectangle nobody
// looks at closely.
func word(page, start, length int) highlight.Box {
	return highlight.Box{
		Page: page,
		Span: domain.Span{From: start, To: start + length},
		Rect: highlight.Rect{MinX: 0.1, MinY: float32(start) / 1000, MaxX: 0.9, MaxY: 0.2},
	}
}

// run is a span of the prose, by where it begins and how far it reaches.
func run(start, length int) domain.Span {
	return domain.Span{From: start, To: start + length}
}

// read is a document read into words, each five bytes with a space after.
func read(pages, each int) []highlight.Box {
	var out []highlight.Box
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
	over := highlight.Over(boxes, run(6, 12))
	if len(over) != 2 {
		t.Fatalf("%d boxes, want the two words the run covers", len(over))
	}
	if over[0].Page != 0 || over[1].Page != 0 {
		t.Errorf("pages %d and %d", over[0].Page, over[1].Page)
	}
}

func TestARunCrossingAPageIsOnBothOfThem(t *testing.T) {
	// Four words a page, so the fourth word of page one and the first of page
	// two are one run.
	boxes := read(3, 4)

	over := highlight.Over(boxes, run(18, 12))
	if len(over) != 2 {
		t.Fatalf("%d boxes, want the two words the run crosses", len(over))
	}
	if over[0].Page != 0 || over[1].Page != 1 {
		t.Errorf("pages %d and %d", over[0].Page, over[1].Page)
	}
}

func TestAWordTheRunOnlyTouchesIsLit(t *testing.T) {
	boxes := read(1, 3)

	// One byte into the second word and one byte out of it.
	over := highlight.Over(boxes, run(7, 2))
	if len(over) != 1 {
		t.Fatalf("%v", over)
	}
}

func TestARunBetweenTwoWordsLightsNeither(t *testing.T) {
	// The space between the first and second word: no box holds it.
	boxes := read(1, 3)

	if over := highlight.Over(boxes, run(5, 1)); over != nil {
		t.Errorf("the gap lit %v", over)
	}
}

func TestNothingIsAskedForAndNothingIsLit(t *testing.T) {
	boxes := read(2, 2)

	if over := highlight.Over(boxes, run(0, 0)); over != nil {
		t.Errorf("a run of nothing lit %v", over)
	}
	if over := highlight.Over(nil, run(0, 10)); over != nil {
		t.Errorf("a document nobody lit lit %v", over)
	}
	if over := highlight.Over(boxes, run(9000, 10)); over != nil {
		t.Errorf("a run past the end lit %v", over)
	}
}

func TestTheBoxesComeBackInTheOrderTheyAreRead(t *testing.T) {
	boxes := read(4, 2)

	var at []int
	for _, one := range highlight.Over(boxes, run(0, 48)) {
		at = append(at, one.Page)
	}
	if want := []int{0, 0, 1, 1, 2, 2, 3, 3}; !reflect.DeepEqual(at, want) {
		t.Errorf("pages %v, want %v", at, want)
	}
}
