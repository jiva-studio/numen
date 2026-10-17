package engine

import (
	"math"
	"testing"
)

func TestNormalise(t *testing.T) {
	v := []float32{3.0, 4.0}
	normed := Normalise(v)
	if len(normed) != 2 || math.IsNaN(float64(normed[0])) || math.IsNaN(float64(normed[1])) {
		t.Fatalf("expected non-NaN vector, got %v", normed)
	}
	if math.Abs(float64(normed[0])-0.6) > 1e-5 {
		t.Errorf("expected 0.6, got %f", normed[0])
	}
	if math.Abs(float64(normed[1])-0.8) > 1e-5 {
		t.Errorf("expected 0.8, got %f", normed[1])
	}

	zero := []float32{0, 0, 0}
	normedZero := Normalise(zero)
	if normedZero == nil || len(normedZero) != 3 {
		t.Fatalf("expected non-nil slice of length 3, got %v", normedZero)
	}
	for i, val := range normedZero {
		if val != 0 {
			t.Errorf("expected 0 at %d, got %f", i, val)
		}
	}
}

func TestGetAttentionBudget(t *testing.T) {
	if got := getAttentionBudget(1); got != 262144 {
		t.Errorf("getAttentionBudget(1) = %d; want 262144", got)
	}
	if got := getAttentionBudget(2); got != 524288 {
		t.Errorf("getAttentionBudget(2) = %d; want 524288", got)
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

	// Multiple rows, seq=3, dim=10:
	// Row 0 has mask [0, 1, 1] (skipping token 0 via continue)
	// Row 1 has mask [0, 0, 0] (kept=0, returns all zeros)
	dim := 10
	seq := 3
	rows := 2
	multiFlat := make([]float32, rows*seq*dim)
	// row 0, token 1: 2.0
	for d := 0; d < dim; d++ {
		multiFlat[(0*seq+1)*dim+d] = 2.0
	}
	// row 0, token 2: 4.0
	for d := 0; d < dim; d++ {
		multiFlat[(0*seq+2)*dim+d] = 4.0
	}
	// row 0, token 0: 99.0 (should be skipped by mask)
	for d := 0; d < dim; d++ {
		multiFlat[(0*seq+0)*dim+d] = 99.0
	}

	multiMask := []int64{
		0, 1, 1, // row 0
		0, 0, 0, // row 1
	}

	multiOut := meanPool(multiFlat, multiMask, rows, seq, dim)
	if len(multiOut) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(multiOut))
	}

	// Row 0: kept tokens 1 and 2 (sum 6.0 across all dim), unit length
	expectedVal := 1.0 / math.Sqrt(float64(dim))
	for d := 0; d < dim; d++ {
		if math.Abs(float64(multiOut[0][d])-expectedVal) > 1e-5 {
			t.Errorf("multiOut[0][%d] = %f; want %f", d, multiOut[0][d], expectedVal)
		}
	}

	// Row 1: all tokens masked out, must be all zero (and not NaN)
	for d := 0; d < dim; d++ {
		if math.IsNaN(float64(multiOut[1][d])) || multiOut[1][d] != 0 {
			t.Errorf("multiOut[1][%d] = %f; want 0", d, multiOut[1][d])
		}
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
		{127, 512, 128},
		{128, 512, 128},
		{129, 512, 256},
		{150, 512, 256},
		{255, 512, 256},
		{256, 512, 256},
		{257, 512, 512},
		{300, 512, 512},
		{300, 400, 512},
		{512, 512, 512},
		{513, 2048, 1024},
		{600, 1024, 1024},
		{1024, 2048, 1024},
		{2000, 1024, 2000},
	}
	for _, tc := range tests {
		got := roundUpToBucket(tc.seq, tc.limit)
		if got != tc.want {
			t.Errorf("roundUpToBucket(%d, %d) = %d; want %d", tc.seq, tc.limit, got, tc.want)
		}
	}
}
