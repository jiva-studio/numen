package recognition

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// makeBox is one printed line on a page, over a run of the prose.
func makeBox(page, start, length int) domain.HighlightBox {
	return domain.HighlightBox{Page: page, ByteSpan: domain.ByteSpan{From: start, To: start + length}}
}

func TestALineIsKnownByItsPlaceInTheWholeRecognition(t *testing.T) {
	prose := "one two three four "
	boxes := []domain.HighlightBox{
		makeBox(4, 0, 4), makeBox(4, 4, 4),
		makeBox(5, 8, 6), makeBox(5, 14, 5),
	}

	pages := getProofreadPages(prose, boxes)
	if len(pages) != 2 {
		t.Fatalf("%d pages, want the two the boxes were read from", len(pages))
	}
	if pages[0].Number != 4 || pages[1].Number != 5 {
		t.Errorf("pages %d and %d", pages[0].Number, pages[1].Number)
	}

	var numbers []int
	var text []string
	for _, page := range pages {
		for _, line := range page.Lines {
			numbers = append(numbers, line.Number)
			text = append(text, line.Text)
		}
	}
	for i, want := range []int{0, 1, 2, 3} {
		if numbers[i] != want {
			t.Errorf("line %d is numbered %d, want its place in the recognition %d", i, numbers[i], want)
		}
	}
	for i, want := range []string{"one ", "two ", "three ", "four "} {
		if text[i] != want {
			t.Errorf("line %d says %q, want %q", i, text[i], want)
		}
	}
}

func TestBoxesSavedForOtherBytesGiveNothing(t *testing.T) {
	prose := "one two"
	boxes := []domain.HighlightBox{makeBox(1, 0, 4), makeBox(1, 4, 90)}

	if pages := getProofreadPages(prose, boxes); pages != nil {
		t.Errorf("a recognition of other bytes came back as %v", pages)
	}
}

func TestBoxesOutOfReadingOrderGiveNothing(t *testing.T) {
	prose := "one two three "
	boxes := []domain.HighlightBox{makeBox(1, 0, 4), makeBox(2, 4, 4), makeBox(1, 8, 6)}

	if pages := getProofreadPages(prose, boxes); pages != nil {
		t.Errorf("a recognition that went back a page came back as %v", pages)
	}
}

func TestABoxWithNoLengthCarriesNoLine(t *testing.T) {
	prose := "one two "
	boxes := []domain.HighlightBox{makeBox(1, 0, 4), makeBox(1, 4, 0), makeBox(1, 4, 4)}

	pages := getProofreadPages(prose, boxes)
	if len(pages) != 1 {
		t.Fatalf("%d pages, want one", len(pages))
	}
	if len(pages[0].Lines) != 2 {
		t.Errorf("%d lines, want the two that say something", len(pages[0].Lines))
	}
	if pages[0].Lines[1].Number != 2 {
		t.Errorf("line numbered %d, want its place in the recognition 2", pages[0].Lines[1].Number)
	}
}
