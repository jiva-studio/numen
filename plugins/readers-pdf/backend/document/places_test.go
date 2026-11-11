package document

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

func TestGetPlacesMergesFollowingBoxesOfAPage(t *testing.T) {
	boxes := []pdf.HighlightBox{
		{Page: 0, Rect: pdf.Rect{MinX: 0.1, MinY: 0.2, MaxX: 0.3, MaxY: 0.4}},
		{Page: 0, Rect: pdf.Rect{MinX: 0.5, MinY: 0.2, MaxX: 0.7, MaxY: 0.4}},
		{Page: 1, Rect: pdf.Rect{MinX: 0.1, MinY: 0.1, MaxX: 0.2, MaxY: 0.2}},
		{Page: 0, Rect: pdf.Rect{MinX: 0.6, MinY: 0.6, MaxX: 0.8, MaxY: 0.8}},
	}

	got := GetPlaces(boxes)

	want := []Place{
		{Page: 0, Rects: []pdf.Rect{boxes[0].Rect, boxes[1].Rect}},
		{Page: 1, Rects: []pdf.Rect{boxes[2].Rect}},
		{Page: 0, Rects: []pdf.Rect{boxes[3].Rect}},
	}
	if !slices.EqualFunc(got, want, func(a, b Place) bool { return a.Page == b.Page && slices.Equal(a.Rects, b.Rects) }) {
		t.Errorf("places %v, want %v", got, want)
	}
}

func TestGetPlacesOfNoBoxesIsEmpty(t *testing.T) {
	if got := GetPlaces(nil); got == nil || len(got) != 0 {
		t.Errorf("places %#v, want an empty list", got)
	}
}
