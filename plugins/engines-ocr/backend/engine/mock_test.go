package engine

import (
	"context"
	"errors"
	"image"
	"testing"
)

func TestMockRecogniserReturnsTextInsidePage(t *testing.T) {
	page := image.NewGray(image.Rect(0, 0, 200, 100))

	blocks, err := MockRecogniser{}.Recognise(t.Context(), page)

	if err != nil || len(blocks) != 1 || blocks[0].Text != MockText || blocks[0].Depth != 0 || len(blocks[0].Boxes) != 1 {
		t.Fatalf("read %+v (%v)", blocks, err)
	}
	box := blocks[0].Boxes[0]
	wantRect := image.Rect(20, 10, 180, 20)
	if box.Rect != wantRect || box.Range.Start != 0 || box.Range.End != len(MockText) {
		t.Errorf("box = %+v, want Rect=%v Start=0 End=%d", box, wantRect, len(MockText))
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := (MockRecogniser{}).Recognise(ctx, page); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled MockRecogniser returned %v, want context.Canceled", err)
	}
}
