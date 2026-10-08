package recognition

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	read "github.com/getcharzp/go-ocr"

	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
)

// BenchmarkRecognisePage measures multi-region recognition on a single page.
func BenchmarkRecognisePage(b *testing.B) {
	page := image.NewRGBA(image.Rect(0, 0, 1200, 1600))
	makeRecogniser := func(count int) *Recogniser {
		rects := make([]image.Rectangle, count)
		for i := range count {
			rects[i] = image.Rect(10, 10+i*100, 500, 90+i*100)
		}
		return &Recogniser{
			layout: partsOf(rects...),
			body:   map[string]bool{"text": true},
			head:   map[string]int{},
			margin: 10,
			read: func(image.Image) ([]read.RecResult, error) {
				time.Sleep(1 * time.Millisecond)
				return []read.RecResult{
					{Text: "what the page says", Score: 0.9, Box: [4]int{0, 0, 100, 20}},
				}, nil
			},
		}
	}
	ctx := b.Context()

	b.Run("1_Region", func(b *testing.B) {
		r := makeRecogniser(1)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = r.Recognise(ctx, page)
		}
	})

	b.Run("4_Regions", func(b *testing.B) {
		r := makeRecogniser(4)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = r.Recognise(ctx, page)
		}
	})

	b.Run("8_Regions", func(b *testing.B) {
		r := makeRecogniser(8)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = r.Recognise(ctx, page)
		}
	})
}

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

// Multi-region recognition preserves the reading order given by the layout.
func TestMultiRegionPreservesReadingOrder(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 800))

	r := &Recogniser{
		layout: partsOf(
			image.Rect(10, 10, 590, 100),
			image.Rect(10, 110, 590, 250),
			image.Rect(10, 260, 590, 450),
		),
		body: map[string]bool{"text": true},
		head: map[string]int{},
		read: func(img image.Image) ([]read.RecResult, error) {
			h := img.Bounds().Dy()
			switch {
			case h < 120:
				time.Sleep(20 * time.Millisecond)
				return []read.RecResult{{Text: "first section", Score: 0.9, Box: [4]int{0, 0, 100, 20}}}, nil
			case h < 180:
				time.Sleep(10 * time.Millisecond)
				return []read.RecResult{{Text: "second section", Score: 0.9, Box: [4]int{0, 0, 100, 20}}}, nil
			default:
				return []read.RecResult{{Text: "third section", Score: 0.9, Box: [4]int{0, 0, 100, 20}}}, nil
			}
		},
	}

	blocks, err := r.Recognise(t.Context(), page)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 3 {
		t.Fatalf("got %d blocks, want 3", len(blocks))
	}
	want := []string{"first section", "second section", "third section"}
	for i, w := range want {
		if blocks[i].Text != w {
			t.Errorf("block %d text = %q, want %q", i, blocks[i].Text, w)
		}
	}
}

// Context cancellation aborts recognition.
func TestRecogniseContextCancelled(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 800))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	r := &Recogniser{
		layout: partsOf(image.Rect(10, 10, 590, 400), image.Rect(10, 410, 590, 790)),
		body:   map[string]bool{"text": true},
		head:   map[string]int{},
		read: func(image.Image) ([]read.RecResult, error) {
			return []read.RecResult{{Text: "text", Score: 0.9, Box: [4]int{0, 0, 100, 20}}}, nil
		},
	}

	if _, err := r.Recognise(ctx, page); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled recognise returned %v, want context.Canceled", err)
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
