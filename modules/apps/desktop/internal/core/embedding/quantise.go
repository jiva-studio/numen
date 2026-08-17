package embedding

import "math"

// Int8Scale is what one unit of an int8 dimension is worth. It is a constant:
// the same float32 must quantise to the same byte in every run, or a vector
// stored today is comparable with one stored after the next book is added.
//
// 0.4 is the scale a unit-length 1024-dimension vector is quantised at.
const Int8Scale = 0.4

// Bits keeps one bit per dimension: the sign, which is the coarse pass's whole
// question. Dimension i is bit 7-i%8 of byte i/8, so the first dimension is the
// most significant bit of the first byte and a bit string reads left to right.
//
// A dimension that is exactly zero is stored as a zero bit.
func Bits(v []float32) []byte {
	out := make([]byte, (len(v)+7)/8)
	for i, x := range v {
		if x > 0 {
			out[i/8] |= 1 << (7 - uint(i)%8)
		}
	}
	return out
}

// Bytes keeps one byte per dimension, for the rerank. Values beyond what
// Int8Scale reaches are clamped; a unit-length vector puts almost nothing there.
func Bytes(v []float32) []int8 {
	out := make([]int8, len(v))
	for i, x := range v {
		q := math.Round(float64(x) / Int8Scale * 127)
		switch {
		case q > 127:
			q = 127
		case q < -127:
			q = -127
		}
		out[i] = int8(q)
	}
	return out
}

// Floats reads bytes back as float32. The result differs from what was
// quantised by up to half a step, and that difference is what the rerank
// tolerates.
func Floats(q []int8) []float32 {
	out := make([]float32, len(q))
	for i, b := range q {
		out[i] = float32(float64(b) / 127 * Int8Scale)
	}
	return out
}

// Normalise scales a vector to unit length, in place, and returns it. A vector
// of all zeros is returned unchanged: there is no direction to keep.
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
