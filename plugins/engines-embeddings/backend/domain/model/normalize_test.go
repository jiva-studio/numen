package model

import (
	"math"
	"testing"
)

func TestNormalize(t *testing.T) {
	v := []float32{3.0, 4.0}
	normed := Normalize(v)
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
	normedZero := Normalize(zero)
	if normedZero == nil || len(normedZero) != 3 {
		t.Fatalf("expected non-nil slice of length 3, got %v", normedZero)
	}
	for i, val := range normedZero {
		if val != 0 {
			t.Errorf("expected 0 at %d, got %f", i, val)
		}
	}
}
