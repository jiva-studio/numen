package recognition

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// box is one printed line on a page, over a run of the prose.
func box(page, start, length int) domain.HighlightBox {
	return domain.HighlightBox{Page: page, ByteSpan: domain.ByteSpan{From: start, To: start + length}}
}

func TestALineIsKnownByItsPlaceInTheWholeRecognition(t *testing.T) {
	prose := "one two three four "
	boxes := []domain.HighlightBox{
		box(4, 0, 4), box(4, 4, 4),
		box(5, 8, 6), box(5, 14, 5),
	}

	batches := getPageGroups(prose, boxes)
	if len(batches) != 2 {
		t.Fatalf("%d batches, want the two pages the boxes were read from", len(batches))
	}
	if batches[0].Number != 4 || batches[1].Number != 5 {
		t.Errorf("pages %d and %d", batches[0].Number, batches[1].Number)
	}

	var numbers []int
	var text []string
	for _, batch := range batches {
		for _, line := range batch.Lines {
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

func TestBoxesWrittenForOtherBytesGiveNothing(t *testing.T) {
	prose := "one two"
	boxes := []domain.HighlightBox{box(1, 0, 4), box(1, 4, 90)}

	if batches := getPageGroups(prose, boxes); batches != nil {
		t.Errorf("a recognition of other bytes came back as %v", batches)
	}
}

// A batch is asked about and answered for by its page. Two batches under one
// page would take one reply between them, and one page's corrections would be
// written onto the other page's lines.
func TestBoxesOutOfReadingOrderGiveNothing(t *testing.T) {
	prose := "one two three "
	boxes := []domain.HighlightBox{box(1, 0, 4), box(2, 4, 4), box(1, 8, 6)}

	batches := getPageGroups(prose, boxes)
	if batches != nil {
		t.Errorf("a recognition that went back a page came back as %v", batches)
	}
	seen := map[int]bool{}
	for _, batch := range batches {
		if seen[batch.Number] {
			t.Errorf("two batches are numbered %d, and one reply keyed by it", batch.Number)
		}
		seen[batch.Number] = true
	}
}

func TestABoxWithNoLengthCarriesNoLine(t *testing.T) {
	prose := "one two "
	boxes := []domain.HighlightBox{box(1, 0, 4), box(1, 4, 0), box(1, 4, 4)}

	batches := getPageGroups(prose, boxes)
	if len(batches) != 1 {
		t.Fatalf("%d batches, want one", len(batches))
	}
	if len(batches[0].Lines) != 2 {
		t.Errorf("%d lines, want the two that say something", len(batches[0].Lines))
	}
	if batches[0].Lines[1].Number != 2 {
		t.Errorf("line numbered %d, want its place in the recognition 2", batches[0].Lines[1].Number)
	}
}
