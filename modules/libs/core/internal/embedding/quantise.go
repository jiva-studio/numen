package embedding

import (
	"math"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

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

// Bytes keeps one byte per dimension, for the rerank, on the grid
// port.Int8Scale sets. Values beyond what that scale reaches are clamped, and a
// clamped dimension is the one loss quantisation here can do that rounding
// cannot undo.
func Bytes(v []float32) []int8 {
	out := make([]int8, len(v))
	for i, x := range v {
		q := math.Round(float64(x) / port.Int8Scale * 127)
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

// Dimensions reads a stored vector back as the dimensions it holds, one per
// byte. It is what Bytes wrote, arriving from storage as unsigned.
func Dimensions(stored []byte) []int8 {
	out := make([]int8, len(stored))
	for i, b := range stored {
		out[i] = int8(b)
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
	if len(query) != len(stored) || len(query) == 0 {
		return 0
	}
	var dot0, dot1, dot2, dot3 float32
	var left0, left1, left2, left3 float32
	var right0, right1, right2, right3 int32

	n := len(stored)
	i := 0
	for ; i+3 < n; i += 4 {
		q0, s0 := query[i], int32(stored[i])
		q1, s1 := query[i+1], int32(stored[i+1])
		q2, s2 := query[i+2], int32(stored[i+2])
		q3, s3 := query[i+3], int32(stored[i+3])

		dot0 += q0 * float32(s0)
		left0 += q0 * q0
		right0 += s0 * s0

		dot1 += q1 * float32(s1)
		left1 += q1 * q1
		right1 += s1 * s1

		dot2 += q2 * float32(s2)
		left2 += q2 * q2
		right2 += s2 * s2

		dot3 += q3 * float32(s3)
		left3 += q3 * q3
		right3 += s3 * s3
	}

	dot := float64((dot0 + dot1) + (dot2 + dot3))
	left := float64((left0 + left1) + (left2 + left3))
	right := float64((right0 + right1) + (right2 + right3))

	for ; i < n; i++ {
		q, s := query[i], int32(stored[i])
		dot += float64(q * float32(s))
		left += float64(q * q)
		right += float64(s * s)
	}

	if left == 0 || right == 0 {
		return 0
	}
	return dot / math.Sqrt(left*right)
}
