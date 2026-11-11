package recognition

import (
	"image"
	"strings"
	"testing"
)

func TestNewPageSplitsBlocksByBox(t *testing.T) {
	bounds := image.Rect(0, 0, 100, 200)
	blocks := []Block{
		{Text: "Chapter One", IsHeading: true, Boxes: []Box{{Rect: image.Rect(10, 0, 90, 20), Range: ByteRange{End: 11}}}},
		{Text: ""},
		{Text: "first line second line", Boxes: []Box{
			{Rect: image.Rect(0, 100, 100, 150), Range: ByteRange{Start: 11, End: 22}},
			{Rect: image.Rect(0, 50, 100, 100), Range: ByteRange{Start: 0, End: 10}},
		}},
		{Text: "a caption nothing placed"},
	}

	parts := newPage(bounds, blocks)

	var joined strings.Builder
	for _, one := range parts {
		joined.WriteString(one.Text)
	}
	if want := "Chapter One\n\nfirst line second line\n\na caption nothing placed\n\n"; joined.String() != want {
		t.Fatalf("the page reads %q, want %q", joined.String(), want)
	}
	if len(parts) != 4 || parts[0].Section != "Chapter One" || parts[1].Section != "" {
		t.Fatalf("parts %+v, want the heading to open a section and each line a part", parts)
	}
	if parts[1].Text != "first line " || parts[1].Rects[0] != [4]float32{0, 0.25, 1, 0.5} {
		t.Errorf("the first line is %+v, want it in the box that begins it", parts[1])
	}
	if parts[2].Rects[0] != [4]float32{0, 0.5, 1, 0.75} || len(parts[3].Rects) != 0 {
		t.Errorf("the second line is %+v and the caption %+v", parts[2], parts[3])
	}
}

func TestGetFractionsOfEmptyPage(t *testing.T) {
	if got := getFractions(image.Rectangle{}, image.Rect(0, 0, 5, 5)); got != [4]float32{} {
		t.Fatalf("got %v", got)
	}
}
