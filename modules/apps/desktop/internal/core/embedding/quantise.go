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

// Coarse is the one bit per dimension of a vector that has been quantised. It
// is read out of the bytes that are stored, so a vector bought and a vector
// reclaimed from the index carry the same bits.
//
// A dimension that quantised to zero is stored as a zero bit.
func Coarse(q []int8) []byte {
	out := make([]byte, (len(q)+7)/8)
	for i, x := range q {
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

// Similarity is the cosine of the angle between a query vector and a stored
// one. It is what the rerank orders by, and the unit a similarity floor is
// stated in.
//
// Each side is divided by its own length, so the scale a stored vector was
// quantised at does not enter. Vectors of different widths, and a vector with
// no direction, are not comparable and answer zero.
func Similarity(query []float32, stored []int8) float64 {
	if len(query) != len(stored) {
		return 0
	}
	var dot, left, right float64
	for i, b := range stored {
		q, s := float64(query[i]), float64(b)
		dot += q * s
		left += q * q
		right += s * s
	}
	if left == 0 || right == 0 {
		return 0
	}
	return dot / math.Sqrt(left*right)
}
