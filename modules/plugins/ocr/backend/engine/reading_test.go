package engine

import (
	"context"
	"errors"
	"image"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	read "github.com/getcharzp/go-ocr"
)

// BenchmarkCropRegion measures cropping a region from a page.
func BenchmarkCropRegion(b *testing.B) {
	page := image.NewRGBA(image.Rect(0, 0, 1200, 1600))
	rect := image.Rect(100, 200, 1100, 800)
	margin := 10

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		crop, _ := cropRegion(page, rect, margin)
		_ = crop
	}
}

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

func TestOnePartThatCouldNotBeReadLeavesTheRest(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 800))
	var asked atomic.Int32

	r := &Recogniser{
		layout: partsOf(image.Rect(10, 10, 590, 400), image.Rect(10, 410, 590, 790)),
		body:   map[string]bool{"text": true},
		head:   map[string]int{},
		read: func(image.Image) ([]read.RecResult, error) {
			if asked.Add(1) == 1 {
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

func partsOf(rects ...image.Rectangle) func(image.Image) ([]Region, error) {
	return func(image.Image) ([]Region, error) {
		out := make([]Region, 0, len(rects))
		for _, rect := range rects {
			out = append(out, Region{Label: "text", Rect: rect})
		}
		return out, nil
	}
}

func TestCropRegionWithSubImage(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 800))
	rect := image.Rect(50, 60, 200, 300)
	margin := 10

	crop, corner := cropRegion(page, rect, margin)
	if crop == nil {
		t.Fatal("crop returned nil")
	}
	wantRect := image.Rect(40, 50, 210, 310)
	if crop.Bounds() != wantRect {
		t.Errorf("crop bounds = %v, want %v", crop.Bounds(), wantRect)
	}
	if corner != wantRect.Min {
		t.Errorf("corner = %v, want %v", corner, wantRect.Min)
	}
}

func TestCropRegionFallback(t *testing.T) {
	type wrapper struct{ image.Image }
	page := wrapper{image.NewRGBA(image.Rect(0, 0, 600, 800))}
	rect := image.Rect(50, 60, 200, 300)
	margin := 10

	crop, corner := cropRegion(page, rect, margin)
	if crop == nil {
		t.Fatal("crop returned nil")
	}
	wantBounds := image.Rect(0, 0, 170, 260)
	if crop.Bounds() != wantBounds {
		t.Errorf("fallback crop bounds = %v, want %v", crop.Bounds(), wantBounds)
	}
	if corner != image.Pt(40, 50) {
		t.Errorf("corner = %v, want (40, 50)", corner)
	}
}

func TestCropRegionEmpty(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 100, 100))
	rect := image.Rect(200, 200, 300, 300)

	crop, corner := cropRegion(page, rect, 0)
	if crop != nil || corner != (image.Point{}) {
		t.Errorf("empty crop returned %v, %v; want nil, (0,0)", crop, corner)
	}
}

func labelled(parts ...Region) func(image.Image) ([]Region, error) {
	return func(image.Image) ([]Region, error) { return parts, nil }
}

func oneLine(text string) func(image.Image) ([]read.RecResult, error) {
	return func(image.Image) ([]read.RecResult, error) {
		return []read.RecResult{{Text: text, Score: 0.9, Box: [4]int{0, 0, 100, 20}}}, nil
	}
}

func TestALayoutThatFailsIsThePagesFailure(t *testing.T) {
	broken := errors.New("the layout model is gone")
	r := &Recogniser{
		layout: func(image.Image) ([]Region, error) { return nil, broken },
		body:   map[string]bool{"text": true},
		read:   oneLine("what the page says"),
	}

	blocks, err := r.Recognise(t.Context(), image.NewRGBA(image.Rect(0, 0, 600, 800)))
	if !errors.Is(err, broken) {
		t.Errorf("answered with %v", err)
	}
	if blocks != nil {
		t.Errorf("blocks = %+v", blocks)
	}
}

func TestAPartOfAnUnwantedKindIsNotRead(t *testing.T) {
	var mu sync.Mutex
	var heights []int
	r := &Recogniser{
		layout: labelled(
			Region{Label: "header", Rect: image.Rect(10, 10, 590, 60)},
			Region{Label: "text", Rect: image.Rect(10, 100, 590, 400)},
		),
		body: map[string]bool{"text": true},
		read: func(img image.Image) ([]read.RecResult, error) {
			mu.Lock()
			heights = append(heights, img.Bounds().Dy())
			mu.Unlock()
			return []read.RecResult{{Text: "what the page says", Score: 0.9, Box: [4]int{0, 0, 100, 20}}}, nil
		},
	}

	blocks, err := r.Recognise(t.Context(), image.NewRGBA(image.Rect(0, 0, 600, 800)))
	if err != nil {
		t.Fatal(err)
	}
	if len(heights) != 1 || heights[0] != 300 {
		t.Errorf("parts read, by height: %v", heights)
	}
	if len(blocks) != 1 || blocks[0].Label != "text" {
		t.Errorf("blocks = %+v", blocks)
	}
}

func TestAPartOutsideThePageIsNotRead(t *testing.T) {
	var mu sync.Mutex
	var asked []image.Image
	r := &Recogniser{
		layout: partsOf(image.Rect(700, 900, 800, 950), image.Rect(10, 10, 590, 400)),
		body:   map[string]bool{"text": true},
		read: func(img image.Image) ([]read.RecResult, error) {
			mu.Lock()
			asked = append(asked, img)
			mu.Unlock()
			return []read.RecResult{{Text: "what the page says", Score: 0.9, Box: [4]int{0, 0, 100, 20}}}, nil
		},
	}

	blocks, err := r.Recognise(t.Context(), image.NewRGBA(image.Rect(0, 0, 600, 800)))
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0] == nil {
		t.Errorf("parts read: %v", asked)
	}
	if len(blocks) != 1 {
		t.Errorf("blocks = %+v", blocks)
	}
}

func TestAPageWithNothingToReadSaysNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	r := &Recogniser{
		layout: labelled(Region{Label: "header", Rect: image.Rect(10, 10, 590, 60)}),
		body:   map[string]bool{"text": true},
		read:   oneLine("never asked"),
	}

	blocks, err := r.Recognise(ctx, image.NewRGBA(image.Rect(0, 0, 600, 800)))
	if err != nil || blocks != nil {
		t.Errorf("answered with %+v, %v", blocks, err)
	}
}

func TestABoxIsAddressedFromThePage(t *testing.T) {
	r := &Recogniser{
		layout: partsOf(image.Rect(50, 60, 590, 400)),
		body:   map[string]bool{"text": true},
		read: func(image.Image) ([]read.RecResult, error) {
			return []read.RecResult{{Text: "what the page says", Score: 0.9, Box: [4]int{3, 5, 40, 70}}}, nil
		},
	}

	blocks, err := r.Recognise(t.Context(), image.NewRGBA(image.Rect(0, 0, 600, 800)))
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || len(blocks[0].Boxes) != 1 {
		t.Fatalf("blocks = %+v", blocks)
	}
	if got, want := blocks[0].Boxes[0].Rect, image.Rect(53, 65, 90, 130); got != want {
		t.Errorf("box = %v, want %v", got, want)
	}
}

func TestAPageOfOnePartThatCouldNotBeReadIsAFailure(t *testing.T) {
	broken := errors.New("the session is gone")
	r := &Recogniser{
		layout: partsOf(image.Rect(10, 10, 590, 400)),
		body:   map[string]bool{"text": true},
		read:   func(image.Image) ([]read.RecResult, error) { return nil, broken },
	}

	if _, err := r.Recognise(t.Context(), image.NewRGBA(image.Rect(0, 0, 600, 800))); !errors.Is(err, broken) {
		t.Errorf("answered with %v", err)
	}
}

func TestAPageOfPartsThatSayNothingIsBlank(t *testing.T) {
	r := &Recogniser{
		layout: partsOf(image.Rect(10, 10, 590, 400)),
		body:   map[string]bool{"text": true},
		read:   func(image.Image) ([]read.RecResult, error) { return nil, nil },
	}

	blocks, err := r.Recognise(t.Context(), image.NewRGBA(image.Rect(0, 0, 600, 800)))
	if err != nil || len(blocks) != 0 {
		t.Errorf("answered with %+v, %v", blocks, err)
	}
}
