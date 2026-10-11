package recognition

import (
	"context"
	"errors"
	"image"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	read "github.com/getcharzp/go-ocr"
)

// gauge records how many calls are in flight and the most there were at once.
type gauge struct {
	current atomic.Int32
	peak    atomic.Int32
}

func (g *gauge) enter() {
	now := g.current.Add(1)
	for {
		seen := g.peak.Load()
		if now <= seen || g.peak.CompareAndSwap(seen, now) {
			return
		}
	}
}

func (g *gauge) leave() { g.current.Add(-1) }

func TestRunBoundedNeverExceedsTheLimit(t *testing.T) {
	for _, tc := range []struct{ limit, count, wantPeak int }{
		{1, 12, 1},
		{3, 12, 3},
		{8, 3, 3},
		{0, 5, 1},
	} {
		var g gauge
		var calls atomic.Int32
		barrier := make(chan struct{})
		var once sync.Once
		go func() {
			time.Sleep(50 * time.Millisecond)
			once.Do(func() { close(barrier) })
		}()
		runBounded(t.Context(), tc.count, tc.limit, func(int) {
			g.enter()
			defer g.leave()
			calls.Add(1)
			if g.current.Load() >= int32(tc.wantPeak) {
				once.Do(func() { close(barrier) })
			}
			<-barrier
		})
		once.Do(func() { close(barrier) })
		if got := g.peak.Load(); got != int32(tc.wantPeak) {
			t.Errorf("limit %d, count %d: peak %d, want %d", tc.limit, tc.count, got, tc.wantPeak)
		}
		if got := calls.Load(); got != int32(tc.count) {
			t.Errorf("limit %d, count %d: %d calls", tc.limit, tc.count, got)
		}
	}
}

func TestRunBoundedWithNoJobsReturns(t *testing.T) {
	runBounded(t.Context(), 0, 4, func(int) { t.Error("called") })
}

func TestRunBoundedStopsStartingAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	var calls atomic.Int32
	runBounded(ctx, 50, 2, func(int) {
		if calls.Add(1) == 3 {
			cancel()
		}
	})
	if got := calls.Load(); got > 4 {
		t.Errorf("%d calls started, want at most one per worker after the third", got)
	}
}

func TestRunBoundedLeavesNoGoroutines(t *testing.T) {
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(t.Context())
	runBounded(ctx, 40, 5, func(i int) {
		if i == 10 {
			cancel()
		}
	})
	runBounded(t.Context(), 40, 5, func(int) {})
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("%d goroutines before, %d after", before, after)
	}
}

func regionRects(n int) []image.Rectangle {
	rects := make([]image.Rectangle, n)
	for i := range rects {
		rects[i] = image.Rect(10, 10+i*20, 590, 25+i*20)
	}
	return rects
}

func TestRecogniseReadsNoMoreRegionsThanJobs(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 800))
	for _, jobs := range []int{1, 3} {
		var g gauge
		r := &Recogniser{
			layout: partsOf(regionRects(9)...),
			body:   map[string]bool{"text": true},
			head:   map[string]int{},
			jobs:   jobs,
			read: func(image.Image) ([]read.RecResult, error) {
				g.enter()
				defer g.leave()
				time.Sleep(20 * time.Millisecond)
				return []read.RecResult{{Text: "text", Score: 0.9, Box: [4]int{0, 0, 10, 10}}}, nil
			},
		}
		blocks, err := r.Recognise(t.Context(), page)
		if err != nil {
			t.Fatal(err)
		}
		if len(blocks) != 9 {
			t.Errorf("jobs %d: %d blocks, want 9", jobs, len(blocks))
		}
		if got := g.peak.Load(); got != int32(jobs) {
			t.Errorf("jobs %d: peak concurrent reads %d", jobs, got)
		}
	}
}

func TestRecogniseKeepsOrderWithFewJobs(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 800))
	rects := regionRects(8)
	r := &Recogniser{
		layout: partsOf(rects...),
		body:   map[string]bool{"text": true},
		head:   map[string]int{},
		jobs:   3,
		read: func(img image.Image) ([]read.RecResult, error) {
			// The crop's top edge identifies the region.
			time.Sleep(time.Duration(8-(img.Bounds().Min.Y/20)) * time.Millisecond)
			return []read.RecResult{{Text: string(rune('a' + img.Bounds().Min.Y/20)), Score: 0.9, Box: [4]int{0, 0, 10, 10}}}, nil
		},
	}
	blocks, err := r.Recognise(t.Context(), page)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != len(rects) {
		t.Fatalf("%d blocks, want %d", len(blocks), len(rects))
	}
	for i, b := range blocks {
		if want := string(rune('a' + i)); b.Text != want {
			t.Errorf("block %d is %q, want %q", i, b.Text, want)
		}
	}
}

func TestRecogniseStopsStartingRegionsOnCancel(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 800))
	ctx, cancel := context.WithCancel(t.Context())
	var started atomic.Int32
	r := &Recogniser{
		layout: partsOf(regionRects(30)...),
		body:   map[string]bool{"text": true},
		head:   map[string]int{},
		jobs:   2,
		read: func(image.Image) ([]read.RecResult, error) {
			if started.Add(1) == 2 {
				cancel()
			}
			return nil, nil
		},
	}
	if _, err := r.Recognise(ctx, page); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if got := started.Load(); got > 4 {
		t.Errorf("%d regions started after cancel", got)
	}
}

func TestRecogniseHandlesOneAndNoRegions(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 800))
	for _, n := range []int{0, 1} {
		r := &Recogniser{
			layout: partsOf(regionRects(n)...),
			body:   map[string]bool{"text": true},
			head:   map[string]int{},
			jobs:   4,
			read: func(image.Image) ([]read.RecResult, error) {
				return []read.RecResult{{Text: "text", Score: 0.9, Box: [4]int{0, 0, 10, 10}}}, nil
			},
		}
		blocks, err := r.Recognise(t.Context(), page)
		if err != nil {
			t.Fatal(err)
		}
		if len(blocks) != n {
			t.Errorf("%d regions gave %d blocks", n, len(blocks))
		}
	}
}
