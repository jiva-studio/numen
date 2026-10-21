package engine

import (
	"math"
)

// Normalise scales a vector to unit length, in place, and returns it. A vector
// of all zeros is returned unchanged.
func Normalise(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return v
	}
	norm := math.Sqrt(sum)
	for i, x := range v {
		v[i] = float32(float64(x) / norm)
	}
	return v
}
