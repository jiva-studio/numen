package engine

import (
	"image"
	"testing"
)

func TestTheMockReadsItsTextInABoxInsideEveryPage(t *testing.T) {
	page := image.NewGray(image.Rect(0, 0, 200, 100))

	blocks, err := MockRecogniser{}.Recognise(t.Context(), page)

	if err != nil || len(blocks) != 1 || blocks[0].Text != MockText || len(blocks[0].Boxes) != 1 {
		t.Fatalf("read %+v (%v)", blocks, err)
	}
	box := blocks[0].Boxes[0]
	if !box.Rect.In(page.Bounds()) || box.Rect.Empty() || box.Span.To != len(MockText) {
		t.Errorf("the box %+v does not cover the text inside the page", box)
	}
}
