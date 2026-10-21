package engine

import (
	"context"
	"errors"
	"image"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
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
	var request atomic.Int32

	r := &Recogniser{
		layout: partsOf(image.Rect(10, 10, 590, 400), image.Rect(10, 410, 590, 790)),
		body:   map[string]bool{"text": true},
		head:   map[string]int{},
		read: func(image.Image) ([]read.RecResult, error) {
			if request.Add(1) == 1 {
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
	var request []image.Image
	r := &Recogniser{
		layout: partsOf(image.Rect(700, 900, 800, 950), image.Rect(10, 10, 590, 400)),
		body:   map[string]bool{"text": true},
		read: func(img image.Image) ([]read.RecResult, error) {
			mu.Lock()
			request = append(request, img)
			mu.Unlock()
			return []read.RecResult{{Text: "what the page says", Score: 0.9, Box: [4]int{0, 0, 100, 20}}}, nil
		},
	}

	blocks, err := r.Recognise(t.Context(), image.NewRGBA(image.Rect(0, 0, 600, 800)))
	if err != nil {
		t.Fatal(err)
	}
	if len(request) != 1 || request[0] == nil {
		t.Errorf("parts read: %v", request)
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

func TestRunBoundedLimitOneCapsWorkersAtOne(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var active atomic.Int32
		var maxActive atomic.Int32
		secondEntered := make(chan struct{})
		isDone := make(chan struct{})

		runBounded(t.Context(), 10, 1, func(int) {
			cur := active.Add(1)
			for {
				old := maxActive.Load()
				if cur <= old || maxActive.CompareAndSwap(old, cur) {
					break
				}
			}
			if cur > 1 {
				select {
				case secondEntered <- struct{}{}:
				default:
				}
			} else {
				select {
				case <-secondEntered:
				case <-time.After(10 * time.Millisecond):
				case <-isDone:
				}
			}
			active.Add(-1)
		})
		close(isDone)
		if got := maxActive.Load(); got != 1 {
			t.Errorf("concurrent workers = %d, want 1", got)
		}
	})
}

func TestDistinctSelectsHigherScoringRegionOnOverlap(t *testing.T) {
	high := Region{Label: "high", Score: 0.9, Rect: image.Rect(0, 0, 100, 100), Order: 0}
	low := Region{Label: "low", Score: 0.5, Rect: image.Rect(10, 10, 90, 90), Order: 1}

	result := Distinct([]Region{low, high}, 0.5)
	if len(result) != 1 || result[0].Label != "high" {
		t.Fatalf("Distinct kept %+v, want only high-scoring region", result)
	}
}

func TestDistinctPreservesInputOrderForEqualScores(t *testing.T) {
	first := Region{Label: "first", Score: 0.8, Rect: image.Rect(0, 0, 100, 100), Order: 0}
	second := Region{Label: "second", Score: 0.8, Rect: image.Rect(10, 10, 90, 90), Order: 1}

	result1 := Distinct([]Region{first, second}, 0.5)
	if len(result1) != 1 || result1[0].Label != "first" {
		t.Fatalf("Distinct kept %+v, want first", result1)
	}

	result2 := Distinct([]Region{second, first}, 0.5)
	if len(result2) != 1 || result2[0].Label != "second" {
		t.Fatalf("Distinct kept %+v, want second", result2)
	}
}

func TestDistinctPreservesDisjointRegionsInOriginalOrder(t *testing.T) {
	tests := []struct {
		name    string
		regions []Region
	}{
		{
			name: "HigherScoreFirstInOrder",
			regions: []Region{
				{Label: "second", Score: 0.5, Rect: image.Rect(60, 60, 100, 100), Order: 2},
				{Label: "first", Score: 0.9, Rect: image.Rect(0, 0, 50, 50), Order: 1},
			},
		},
		{
			name: "LowerScoreFirstInOrder",
			regions: []Region{
				{Label: "first", Score: 0.2, Rect: image.Rect(0, 0, 50, 50), Order: 1},
				{Label: "second", Score: 0.9, Rect: image.Rect(60, 60, 100, 100), Order: 2},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Distinct(tc.regions, 0.5)
			if len(result) != 2 || result[0].Label != "first" || result[1].Label != "second" {
				t.Fatalf("Distinct kept %+v, want [first, second] sorted by order", result)
			}
		})
	}
}

func TestAssembleMultipleLinesAndHyphenation(t *testing.T) {
	lines := []Line{
		{Text: "anti-", Box: image.Rect(50, 10, 100, 30)},
		{Text: "the", Box: image.Rect(0, 10, 45, 30)},
		{Text: "gravity.", Box: image.Rect(0, 40, 60, 60)},
	}

	text, boxes := Assemble(lines)
	wantText := "the antigravity."
	if text != wantText {
		t.Errorf("Assemble text = %q, want %q", text, wantText)
	}
	if len(boxes) != 3 {
		t.Fatalf("got %d boxes, want 3", len(boxes))
	}
	if got, want := text[boxes[0].Span.From:boxes[0].Span.To], "the"; got != want {
		t.Errorf("box 0 span text = %q, want %q", got, want)
	}
	if got, want := text[boxes[1].Span.From:boxes[1].Span.To], "anti"; got != want {
		t.Errorf("box 1 span text = %q, want %q", got, want)
	}
	if got, want := text[boxes[2].Span.From:boxes[2].Span.To], "gravity."; got != want {
		t.Errorf("box 2 span text = %q, want %q", got, want)
	}
	if boxes[2].Span.From < boxes[1].Span.To {
		t.Errorf("box 2 span from %d < box 1 span to %d", boxes[2].Span.From, boxes[1].Span.To)
	}
}

func TestAssembleSortsLinesVertically(t *testing.T) {
	lines := []Line{
		{Text: "second", Box: image.Rect(0, 50, 100, 70)},
		{Text: "first", Box: image.Rect(0, 10, 100, 30)},
	}
	text, _ := Assemble(lines)
	if text != "first second" {
		t.Errorf("text = %q, want %q", text, "first second")
	}
}

func TestAssembleSingleCharacterWords(t *testing.T) {
	lines := []Line{
		{Text: "a", Box: image.Rect(0, 10, 10, 30)},
		{Text: "b", Box: image.Rect(20, 10, 30, 30)},
		{Text: "c", Box: image.Rect(0, 50, 10, 70)},
	}
	text, boxes := Assemble(lines)
	if text != "a b c" {
		t.Errorf("text = %q, want %q", text, "a b c")
	}
	if len(boxes) != 3 {
		t.Fatalf("got %d boxes, want 3", len(boxes))
	}
	if boxes[0].Span != (ByteSpan{From: 0, To: 1}) ||
		boxes[1].Span != (ByteSpan{From: 2, To: 3}) ||
		boxes[2].Span != (ByteSpan{From: 4, To: 5}) {
		t.Errorf("spans = %+v", boxes)
	}
}

func TestAssembleIgnoresEmptyAndWhitespaceLines(t *testing.T) {
	lines := []Line{
		{Text: "first", Box: image.Rect(0, 10, 50, 30)},
		{Text: "   ", Box: image.Rect(0, 35, 50, 45)},
		{Text: "second", Box: image.Rect(0, 50, 50, 70)},
	}
	text, boxes := Assemble(lines)
	if text != "first second" {
		t.Errorf("text = %q, want %q", text, "first second")
	}
	if len(boxes) != 2 {
		t.Fatalf("got %d boxes, want 2", len(boxes))
	}
}

func TestAssembleNoHyphenAcrossLines(t *testing.T) {
	lines := []Line{
		{Text: "hello", Box: image.Rect(0, 10, 50, 30)},
		{Text: "world", Box: image.Rect(0, 40, 50, 60)},
	}
	text, _ := Assemble(lines)
	if text != "hello world" {
		t.Errorf("text = %q, want %q", text, "hello world")
	}
}

func TestGroupEmptyAndSingleLine(t *testing.T) {
	countEmpty := 0
	for range group(nil) {
		countEmpty++
	}
	if countEmpty != 0 {
		t.Errorf("group(nil) yielded %d lines, want 0", countEmpty)
	}

	var singleGroups [][]Line
	for l := range group([]Line{{Text: "one", Box: image.Rect(0, 0, 10, 10)}}) {
		singleGroups = append(singleGroups, l)
	}
	if len(singleGroups) != 1 || len(singleGroups[0]) != 1 {
		t.Errorf("group(single) yielded %v, want 1 group with 1 line", singleGroups)
	}
}

func TestGroupVerticalOverlapThreshold(t *testing.T) {
	lines := []Line{
		{Text: "l1a", Box: image.Rect(0, 10, 50, 30)},
		{Text: "l1b", Box: image.Rect(60, 15, 110, 35)},
		{Text: "l2", Box: image.Rect(0, 40, 50, 60)},
	}
	var groups [][]Line
	for g := range group(lines) {
		groups = append(groups, g)
	}
	if len(groups) != 2 {
		t.Fatalf("group count = %d, want 2", len(groups))
	}
	if len(groups[0]) != 2 || len(groups[1]) != 1 {
		t.Errorf("groups = %+v", groups)
	}
}

func TestGroupZeroHeightBoxDoesNotYieldEmptySlice(t *testing.T) {
	lines := []Line{
		{Text: "flat", Box: image.Rect(0, 10, 50, 10)},
		{Text: "normal", Box: image.Rect(0, 20, 50, 40)},
	}
	for g := range group(lines) {
		if len(g) == 0 {
			t.Fatal("group yielded an empty line slice")
		}
	}
}

func TestGroupExactFiftyPercentOverlapKeepsSameLine(t *testing.T) {
	lines := []Line{
		{Text: "a", Box: image.Rect(0, 10, 50, 30)},
		{Text: "b", Box: image.Rect(60, 20, 110, 40)},
	}
	var groups [][]Line
	for g := range group(lines) {
		groups = append(groups, g)
	}
	if len(groups) != 1 || len(groups[0]) != 2 {
		t.Fatalf("group count = %d, want 1 group with 2 boxes", len(groups))
	}
}

func TestGroupEarlyTerminationStopsYielding(t *testing.T) {
	lines := []Line{
		{Text: "l1", Box: image.Rect(0, 10, 50, 30)},
		{Text: "l2", Box: image.Rect(0, 40, 50, 60)},
		{Text: "l3", Box: image.Rect(0, 70, 50, 90)},
	}
	count := 0
	for range group(lines) {
		count++
		break
	}
	if count != 1 {
		t.Errorf("yielded %d times after break, want 1", count)
	}
}

func TestAssembleWhitespaceBoxWithinSameLine(t *testing.T) {
	tests := []struct {
		name  string
		lines []Line
		want  string
	}{
		{
			name: "TrailingWhitespaceDropped",
			lines: []Line{
				{Text: "hello", Box: image.Rect(0, 10, 50, 30)},
				{Text: "   ", Box: image.Rect(60, 10, 100, 30)},
			},
			want: "hello",
		},
		{
			name: "LeadingWhitespaceIgnored",
			lines: []Line{
				{Text: "   ", Box: image.Rect(0, 10, 40, 30)},
				{Text: "valid", Box: image.Rect(50, 10, 100, 30)},
			},
			want: "valid",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, boxes := Assemble(tc.lines)
			if text != tc.want {
				t.Errorf("text = %q, want %q", text, tc.want)
			}
			if len(boxes) != 1 {
				t.Fatalf("got %d boxes, want 1", len(boxes))
			}
		})
	}
}

func TestOverlap(t *testing.T) {
	tests := []struct {
		name string
		a, b image.Rectangle
		want float64
	}{
		{
			name: "Disjoint",
			a:    image.Rect(0, 0, 10, 10),
			b:    image.Rect(20, 20, 30, 30),
			want: 0,
		},
		{
			name: "Partial",
			a:    image.Rect(0, 0, 10, 10),
			b:    image.Rect(5, 0, 15, 20),
			want: 0.2,
		},
		{
			name: "UnitArea",
			a:    image.Rect(0, 0, 1, 1),
			b:    image.Rect(0, 0, 1, 1),
			want: 1.0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := overlap(tc.a, tc.b); got != tc.want {
				t.Errorf("overlap(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestIsInsideDisjointRectangles(t *testing.T) {
	a := image.Rect(0, 0, 10, 10)
	b := image.Rect(20, 20, 30, 30)
	if isInside(a, b) {
		t.Error("isInside returned true for disjoint rectangles")
	}
}

func TestAssembleHyphenation(t *testing.T) {
	tests := []struct {
		name      string
		lines     []Line
		wantText  string
		wantSpans []string
	}{
		{
			name: "SingleWordHyphenation",
			lines: []Line{
				{Text: "pre-", Box: image.Rect(0, 10, 50, 30)},
				{Text: "fix", Box: image.Rect(0, 40, 50, 60)},
			},
			wantText:  "prefix",
			wantSpans: []string{"pre", "fix"},
		},
		{
			name: "HyphenAfterNonLetterNotStripped",
			lines: []Line{
				{Text: "clause 1-", Box: image.Rect(0, 10, 80, 30)},
				{Text: "2", Box: image.Rect(0, 40, 50, 60)},
			},
			wantText:  "clause 1- 2",
			wantSpans: []string{"clause 1-", "2"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, boxes := Assemble(tc.lines)
			if text != tc.wantText {
				t.Errorf("Assemble text = %q, want %q", text, tc.wantText)
			}
			if len(boxes) != len(tc.wantSpans) {
				t.Fatalf("got %d boxes, want %d", len(boxes), len(tc.wantSpans))
			}
			for i, wantSpan := range tc.wantSpans {
				if got := text[boxes[i].Span.From:boxes[i].Span.To]; got != wantSpan {
					t.Errorf("box %d span text = %q, want %q", i, got, wantSpan)
				}
			}
		})
	}
}

func TestMeasureVerticalOverlap(t *testing.T) {
	tests := []struct {
		name string
		line []Line
		box  Line
		want float64
	}{
		{
			name: "MultipleLinesTop",
			line: []Line{
				{Box: image.Rect(0, 10, 50, 30)},
				{Box: image.Rect(0, 15, 50, 35)},
			},
			box:  Line{Box: image.Rect(0, 12, 50, 32)},
			want: 0.75,
		},
		{
			name: "VaryingHeight",
			line: []Line{{Box: image.Rect(0, 10, 50, 20)}},
			box:  Line{Box: image.Rect(0, 10, 50, 30)},
			want: 1.0,
		},
		{
			name: "ZeroHeight",
			line: []Line{{Box: image.Rect(0, 10, 50, 30)}},
			box:  Line{Box: image.Rect(0, 10, 50, 10)},
			want: 0,
		},
		{
			name: "UnitHeight",
			line: []Line{{Box: image.Rect(0, 10, 50, 11)}},
			box:  Line{Box: image.Rect(0, 10, 50, 11)},
			want: 1.0,
		},
		{
			name: "DisjointVertically",
			line: []Line{{Box: image.Rect(0, 10, 50, 20)}},
			box:  Line{Box: image.Rect(0, 30, 50, 40)},
			want: 0,
		},
		{
			name: "PartialOverlapArithmetic",
			line: []Line{{Box: image.Rect(0, 10, 50, 30)}},
			box:  Line{Box: image.Rect(0, 20, 50, 40)},
			want: 0.5,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := measureVerticalOverlap(tc.line, tc.box); got != tc.want {
				t.Errorf("overlap = %v, want %v", got, tc.want)
			}
		})
	}
}
