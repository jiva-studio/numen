package recognition

import (
	"errors"
	"image"
	"testing"

	read "github.com/getcharzp/go-ocr"

	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
)

// A page whose every part was refused says nothing, and a page that says
// nothing reads downstream as a blank page. It is a failure and is said to be
// one.
func TestAPageNothingCouldBeReadOnIsAFailure(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 800))
	broken := errors.New("the session is gone")

	r := &Recogniser{
		layout: partsOf(image.Rect(10, 10, 590, 400), image.Rect(10, 410, 590, 790)),
		body:   map[string]bool{"text": true},
		head:   map[string]int{},
		read: func(image.Image) ([]read.RecResult, error) {
			return nil, broken
		},
	}

	if _, err := r.Recognise(t.Context(), page); !errors.Is(err, broken) {
		t.Errorf("a page nothing could be read on answered with %v", err)
	}
}

// One part that could not be read is one part: the rest of the page is still
// what it says.
func TestOnePartThatCouldNotBeReadLeavesTheRest(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 800))
	asked := 0

	r := &Recogniser{
		layout: partsOf(image.Rect(10, 10, 590, 400), image.Rect(10, 410, 590, 790)),
		body:   map[string]bool{"text": true},
		head:   map[string]int{},
		read: func(image.Image) ([]read.RecResult, error) {
			asked++
			if asked == 1 {
				return nil, errors.New("the first part")
			}
			return []read.RecResult{{Text: "what the page says", Score: 0.9, Box: [4]int{0, 0, 100, 20}}}, nil
		},
	}

	blocks, err := r.Recognise(t.Context(), page)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || blocks[0].Text != "what the page says" {
		t.Errorf("the page came back as %+v", blocks)
	}
}

// partsOf is a layout that divides every page the same way.
func partsOf(rects ...image.Rectangle) func(image.Image) ([]ocr.Region, error) {
	return func(image.Image) ([]ocr.Region, error) {
		out := make([]ocr.Region, 0, len(rects))
		for _, rect := range rects {
			out = append(out, ocr.Region{Label: "text", Rect: rect})
		}
		return out, nil
	}
}
