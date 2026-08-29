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
