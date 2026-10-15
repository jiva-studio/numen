package engine

import (
	"math"
	"testing"
)

func TestNormalise(t *testing.T) {
	v := []float32{3.0, 4.0}
	normed := Normalise(v)
	expectedNorm := float64(normed[0]*normed[0] + normed[1]*normed[1])
	if math.Abs(expectedNorm-1.0) > 1e-5 {
		t.Fatalf("expected unit norm 1.0, got %f", expectedNorm)
	}

	zero := []float32{0, 0, 0}
	normedZero := Normalise(zero)
	for i, val := range normedZero {
		if val != 0 {
			t.Errorf("expected 0 at %d, got %f", i, val)
		}
	}
}

func TestHeadPool(t *testing.T) {
	// 2 rows, seq length 2, dim 3
	flat := []float32{
		// row 0, token 0 (CLS / head)
		1.0, 2.0, 2.0,
		// row 0, token 1
		9.0, 9.0, 9.0,
		// row 1, token 0 (CLS / head)
		0.0, 3.0, 4.0,
		// row 1, token 1
		8.0, 8.0, 8.0,
	}
	out := headPool(flat, 2, 2, 3)
	if len(out) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(out))
	}
	if len(out[0]) != 3 || len(out[1]) != 3 {
		t.Fatalf("expected 3 dimensions per vector")
	}

	// Token 0 of row 0: [1, 2, 2], norm = 3 -> [1/3, 2/3, 2/3]
	if math.Abs(float64(out[0][0])-1.0/3.0) > 1e-5 {
		t.Errorf("expected 1/3, got %f", out[0][0])
	}
	// Token 0 of row 1: [0, 3, 4], norm = 5 -> [0, 3/5, 4/5]
	if math.Abs(float64(out[1][1])-3.0/5.0) > 1e-5 {
		t.Errorf("expected 3/5, got %f", out[1][1])
	}
}

func TestMeanPool(t *testing.T) {
	// 1 row, seq length 2, dim 2
	flat := []float32{
		1.0, 2.0,
		3.0, 4.0,
	}
	mask := []int64{1, 1}
	out := meanPool(flat, mask, 1, 2, 2)
	if len(out) != 1 {
		t.Fatalf("expected 1 vector")
	}
	// mean is [2.0, 3.0], norm is sqrt(4+9) = sqrt(13)
	norm := float32(math.Sqrt(13))
	if math.Abs(float64(out[0][0])-float64(2.0/norm)) > 1e-5 {
		t.Errorf("expected %f, got %f", 2.0/norm, out[0][0])
	}

	// test masking out token 1
	maskIgnored := []int64{1, 0}
	outIgnored := meanPool(flat, maskIgnored, 1, 2, 2)
	// only token 0 kept: [1, 2], norm = sqrt(5)
	norm0 := float32(math.Sqrt(5))
	if math.Abs(float64(outIgnored[0][0])-float64(1.0/norm0)) > 1e-5 {
		t.Errorf("expected %f, got %f", 1.0/norm0, outIgnored[0][0])
	}
}

func TestRoundUpToBucket(t *testing.T) {
	tests := []struct {
		seq   int
		limit int
		want  int
	}{
		{10, 512, 64},
		{64, 512, 64},
		{65, 512, 128},
		{150, 512, 256},
		{300, 512, 512},
		{600, 1024, 1024},
		{2000, 1024, 2000},
	}
	for _, tc := range tests {
		got := roundUpToBucket(tc.seq, tc.limit)
		if got != tc.want {
			t.Errorf("roundUpToBucket(%d, %d) = %d; want %d", tc.seq, tc.limit, got, tc.want)
		}
	}
}
