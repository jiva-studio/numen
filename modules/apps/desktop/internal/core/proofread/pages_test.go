package proofread_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/placed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/proofread"
)

// box is one printed line on a page, over a run of the prose.
func box(page, start, length int) placed.Box {
	return placed.Box{Page: page, Start: start, Length: length}
}

func TestALineIsKnownByItsPlaceInTheWholeReading(t *testing.T) {
	prose := "one two three four "
	boxes := []placed.Box{
		box(4, 0, 4), box(4, 4, 4),
		box(5, 8, 6), box(5, 14, 5),
	}

	pages := proofread.Pages(prose, boxes)
	if len(pages) != 2 {
		t.Fatalf("%d pages, want the two the boxes were read from", len(pages))
	}
	if pages[0].At != 4 || pages[1].At != 5 {
		t.Errorf("pages %d and %d", pages[0].At, pages[1].At)
	}

	var numbers []int
	var text []string
	for _, page := range pages {
		for _, line := range page.Lines {
			numbers = append(numbers, line.At)
			text = append(text, line.Text)
		}
	}
	for i, want := range []int{0, 1, 2, 3} {
		if numbers[i] != want {
			t.Errorf("line %d is numbered %d, want its place in the reading %d", i, numbers[i], want)
		}
	}
	for i, want := range []string{"one ", "two ", "three ", "four "} {
		if text[i] != want {
			t.Errorf("line %d says %q, want %q", i, text[i], want)
		}
	}
}

func TestBoxesWrittenForOtherBytesGiveNothing(t *testing.T) {
	prose := "one two"
	boxes := []placed.Box{box(1, 0, 4), box(1, 4, 90)}

	if pages := proofread.Pages(prose, boxes); pages != nil {
		t.Errorf("a reading of other bytes came back as %v", pages)
	}
}

func TestABoxWithNoLengthCarriesNoLine(t *testing.T) {
	prose := "one two "
	boxes := []placed.Box{box(1, 0, 4), box(1, 4, 0), box(1, 4, 4)}

	pages := proofread.Pages(prose, boxes)
	if len(pages) != 1 {
		t.Fatalf("%d pages, want one", len(pages))
	}
	if len(pages[0].Lines) != 2 {
		t.Errorf("%d lines, want the two that say something", len(pages[0].Lines))
	}
	if pages[0].Lines[1].At != 2 {
		t.Errorf("line numbered %d, want its place in the reading 2", pages[0].Lines[1].At)
	}
}
