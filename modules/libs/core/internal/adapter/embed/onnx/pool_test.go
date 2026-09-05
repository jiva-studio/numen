package onnx

import (
	"math"
	"slices"
	"testing"
)

func TestPaddingIsNotAveragedIn(t *testing.T) {
	// Two texts of one token in a batch padded to three. The padded positions
	// hold a value the mask excludes.
	flat := []float32{
		3, 0, 100, 100, 100, 100,
		0, 3, 100, 100, 100, 100,
	}
	mask := [][]int64{{1, 0, 0}, {1, 0, 0}}
	got := meanPool(flat, mask, 2)
	if !slices.Equal(got[0], []float32{1, 0}) || !slices.Equal(got[1], []float32{0, 1}) {
		t.Errorf("got %v", got)
	}
}

func TestTokensAreAveragedAndTheResultIsUnitLength(t *testing.T) {
	flat := []float32{
		1, 0,
		0, 1,
	}
	got := meanPool(flat, [][]int64{{1, 1}}, 2)
	want := float32(math.Sqrt2 / 2)
	if math.Abs(float64(got[0][0]-want)) > 1e-6 || math.Abs(float64(got[0][1]-want)) > 1e-6 {
		t.Errorf("got %v, want %v", got[0], want)
	}
}

func TestTheFirstTokenIsTheVectorWhenTheModelPoolsThatWay(t *testing.T) {
	// Two texts of three tokens. Only the first token of each carries the
	// vector; a model trained this way puts nothing in the rest.
	flat := []float32{
		3, 0, 9, 9, 9, 9,
		0, 5, 9, 9, 9, 9,
	}
	got := headPool(flat, 2, 3, 2)
	if !slices.Equal(got[0], []float32{1, 0}) || !slices.Equal(got[1], []float32{0, 1}) {
		t.Errorf("got %v", got)
	}
}

func TestAllPaddingIsAZeroVector(t *testing.T) {
	got := meanPool([]float32{5, 5}, [][]int64{{0}}, 2)
	if !slices.Equal(got[0], []float32{0, 0}) {
		t.Errorf("got %v", got)
	}
}

func TestSequenceLengthsRoundUpToAStep(t *testing.T) {
	for _, c := range []struct {
		tokens, want int
	}{
		{1, 64},
		{64, 64},
		{65, 128},
		{200, 256},
		{256, 256},
		{9000, 256},
	} {
		if got := bucket(c.tokens, 64, 256); got != c.want {
			t.Errorf("%d tokens gave %d, want %d", c.tokens, got, c.want)
		}
	}
}

// A shape is compiled the first time it appears, so a batch that is not full is
// laid out in as many rows as a full one and padding fills the rest.
func TestABatchIsLaidOutAtTheSizeAFullOneCarries(t *testing.T) {
	const rows, seq = 8, 64
	for texts := 1; texts <= rows; texts++ {
		batch := make([][]int, texts)
		for i := range batch {
			batch[i] = []int{7, 8, 9}
		}
		ids, mask, types := padded(batch, rows, seq, 1)
		for _, held := range [][][]int64{ids, mask, types} {
			if len(held) != rows {
				t.Fatalf("%d texts were laid out in %d rows", texts, len(held))
			}
			for _, row := range held {
				if len(row) != seq {
					t.Fatalf("%d texts gave a row of %d", texts, len(row))
				}
			}
		}
		// A row nothing was written into is padding, and what pools it divides
		// by the tokens it is marked at.
		for row := texts; row < rows; row++ {
			if ids[row][0] != 1 {
				t.Errorf("row %d of %d holds %d", row, texts, ids[row][0])
			}
			var marked int64
			for _, at := range mask[row] {
				marked += at
			}
			if marked != 1 {
				t.Errorf("row %d of %d is marked at %d tokens", row, texts, marked)
			}
		}
	}
}

// The shapes one run meets are the sequence lengths and no more, whatever a
// batch holds.
func TestARunMeetsOneShapePerSequenceLength(t *testing.T) {
	const rows, limit, step = 8, 256, 64
	seen := map[[2]int]bool{}
	for texts := 1; texts <= rows; texts++ {
		for _, tokens := range []int{1, 40, 65, 200, 300} {
			batch := make([][]int, texts)
			for i := range batch {
				batch[i] = make([]int, tokens)
			}
			ids, _, _ := padded(batch, rows, bucket(tokens, step, limit), 1)
			seen[[2]int{len(ids), len(ids[0])}] = true
		}
	}
	if want := limit/step + 2; len(seen) > want {
		t.Errorf("a run met %d shapes, and the cache holds %d", len(seen), want)
	}
}
