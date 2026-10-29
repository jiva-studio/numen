package floor_test

import (
	"math"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/floor"
)

func TestSummarizeAKnownDistribution(t *testing.T) {
	// 0.00, 0.01, ... 1.00: the median is 0.5, the quartiles 0.25 and 0.75.
	sample := make([]float64, 101)
	for i := range sample {
		sample[i] = float64(i) / 100
	}
	got := floor.Summarize(sample)

	wantSpread := 0.5 / 1.349
	if math.Abs(got.Median-0.5) > 1e-9 {
		t.Errorf("median = %v, want 0.5", got.Median)
	}
	if math.Abs(got.Spread-wantSpread) > 1e-9 {
		t.Errorf("spread = %v, want %v", got.Spread, wantSpread)
	}
	// 0.5 + 2*0.3706 is above the upper bound.
	if got.Floor != 0.95 {
		t.Errorf("floor = %v, want the upper bound 0.95", got.Floor)
	}
	if len(got.Histogram) != 20 {
		t.Fatalf("histogram has %d bins, want 20", len(got.Histogram))
	}
	total := 0
	for _, count := range got.Histogram {
		total += count
	}
	if total != 101 {
		t.Errorf("histogram counts %d values, want 101", total)
	}
	if got.Histogram[0] != 5 || got.Histogram[19] != 6 {
		t.Errorf("first and last bins hold %d and %d, want 5 and 6", got.Histogram[0], got.Histogram[19])
	}
}

func TestSummarizeFloorIsMedianPlusTwoSpreads(t *testing.T) {
	// Quartiles 0.25 and 0.35 around a median of 0.3.
	sample := []float64{0.2, 0.25, 0.3, 0.35, 0.4}
	got := floor.Summarize(sample)

	wantSpread := (0.35 - 0.25) / 1.349
	want := 0.3 + 2*wantSpread
	if math.Abs(got.Floor-want) > 1e-9 {
		t.Errorf("floor = %v, want %v", got.Floor, want)
	}
}

func TestSummarizeClampsTheFloorAndTheSpread(t *testing.T) {
	for _, tc := range []struct {
		name       string
		sample     []float64
		wantFloor  float64
		wantSpread float64
	}{
		{"low sample takes the lower bound", []float64{0, 0, 0, 0, 0}, 0.10, 0.01},
		{"identical values take the minimum spread", []float64{0.4, 0.4, 0.4, 0.4}, 0.4 + 2*0.01, 0.01},
		{"high sample takes the upper bound", []float64{0.99, 0.99, 0.99}, 0.95, 0.01},
		{"negative similarities take the lower bound", []float64{-0.5, -0.4, -0.3}, 0.10, 0.1 / 1.349},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := floor.Summarize(tc.sample)
			if math.Abs(got.Floor-tc.wantFloor) > 1e-9 {
				t.Errorf("floor = %v, want %v", got.Floor, tc.wantFloor)
			}
			if math.Abs(got.Spread-tc.wantSpread) > 1e-9 {
				t.Errorf("spread = %v, want %v", got.Spread, tc.wantSpread)
			}
		})
	}
}

func TestSummarizeHistogramClampsToTheRange(t *testing.T) {
	got := floor.Summarize([]float64{-0.3, 0, 0.049, 0.05, 0.5, 0.999, 1, 1.4})
	want := make([]int, 20)
	want[0] = 3  // -0.3, 0, 0.049
	want[1] = 1  // 0.05
	want[10] = 1 // 0.5
	want[19] = 3 // 0.999, 1, 1.4
	if !slices.Equal(got.Histogram, want) {
		t.Errorf("histogram = %v, want %v", got.Histogram, want)
	}
}

func TestSummarizeDoesNotChangeTheSample(t *testing.T) {
	sample := []float64{0.9, 0.1, 0.5}
	floor.Summarize(sample)
	if !slices.Equal(sample, []float64{0.9, 0.1, 0.5}) {
		t.Errorf("the sample was reordered: %v", sample)
	}
}

func TestSummarizeAnEmptySampleHasTheLowerBound(t *testing.T) {
	got := floor.Summarize(nil)
	if got.Floor != 0.10 || len(got.Histogram) != 20 {
		t.Errorf("got %+v", got)
	}
}
