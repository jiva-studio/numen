package onnx

import (
	"math"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
)

// A released padding is reused, and the batch laid out on it holds nothing of
// the one before.
func TestAReusedPaddingHoldsNothingOfThePreviousBatch(t *testing.T) {
	_, _, first := padBatch([][]int{{7, 8, 9, 10}, {5, 6, 7, 8}}, 3)
	first.types[0], first.types[5] = 9, 9
	releasePadding(first)

	rows, seq, in := padBatch([][]int{{4}, {}}, 1)
	if rows != 2 || seq != 1 {
		t.Fatalf("laid out %dx%d", rows, seq)
	}
	if !slices.Equal(in.ids, []int64{4, 1}) || !slices.Equal(in.mask, []int64{1, 1}) || !slices.Equal(in.types, []int64{0, 0}) {
		t.Errorf("ids %v mask %v types %v", in.ids, in.mask, in.types)
	}
	releasePadding(in)

	_, _, wide := padBatch([][]int{{1}, {1, 2, 3}}, 0)
	if !slices.Equal(wide.mask, []int64{1, 0, 0, 1, 1, 1}) || !slices.Equal(wide.ids, []int64{1, 0, 0, 1, 2, 3}) || !slices.Equal(wide.types, make([]int64, 6)) {
		t.Errorf("ids %v mask %v types %v", wide.ids, wide.mask, wide.types)
	}
}

// Unrolling adds along the dimensions, and each dimension still sums its tokens
// in order, so the result is bit-identical to the plain loop.
func TestMeanPoolMatchesThePlainLoopBitForBit(t *testing.T) {
	const rows, seq = 3, 7
	for _, dimensions := range []int{1, 3, 4, 5, 8, 13} {
		flat := make([]float32, rows*seq*dimensions)
		for i := range flat {
			flat[i] = float32(math.Sin(float64(i))) * 1e3
		}
		mask := make([]int64, rows*seq)
		for i := range mask {
			mask[i] = int64(i % 3 % 2)
		}
		got := meanPool(flat, mask, rows, seq, dimensions)
		for row := range rows {
			want := make([]float32, dimensions)
			kept := 0
			for token := range seq {
				if mask[row*seq+token] == 0 {
					continue
				}
				kept++
				for d := range dimensions {
					want[d] += flat[(row*seq+token)*dimensions+d]
				}
			}
			if kept > 0 {
				for d := range want {
					want[d] /= float32(kept)
				}
			}
			want = embedding.Normalise(want)
			if !slices.Equal(got[row], want) {
				t.Errorf("dimensions %d row %d: got %v want %v", dimensions, row, got[row], want)
			}
		}
	}
}
