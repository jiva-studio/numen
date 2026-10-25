package recognition

import (
	"image"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
)

func TestNewPageSplitsBlocksByBox(t *testing.T) {
	bounds := image.Rect(0, 0, 100, 200)
	blocks := []engine.Block{
		{Text: "Chapter One", IsHeading: true, Boxes: []engine.Box{{Rect: image.Rect(10, 0, 90, 20), Span: engine.ByteSpan{To: 11}}}},
		{Text: ""},
		{Text: "first line second line", Boxes: []engine.Box{
			{Rect: image.Rect(0, 100, 100, 150), Span: engine.ByteSpan{From: 11, To: 22}},
			{Rect: image.Rect(0, 50, 100, 100), Span: engine.ByteSpan{From: 0, To: 10}},
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
