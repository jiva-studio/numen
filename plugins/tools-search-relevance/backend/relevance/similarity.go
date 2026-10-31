package relevance

import "math"

// getSimilarity is the cosine of the angle between a query vector and a stored vector of signed bytes.
// The four partial sums are kept apart so that the result is the same for the same vectors on every machine.
func getSimilarity(query []float32, stored []byte) float64 {
	if len(query) != len(stored) || len(query) == 0 {
		return 0
	}
	var dot0, dot1, dot2, dot3 float32
	var left0, left1, left2, left3 float32
	var right0, right1, right2, right3 int32

	n := len(stored)
	i := 0
	for ; i+3 < n; i += 4 {
		q0, s0 := query[i], int32(int8(stored[i]))
		q1, s1 := query[i+1], int32(int8(stored[i+1]))
		q2, s2 := query[i+2], int32(int8(stored[i+2]))
		q3, s3 := query[i+3], int32(int8(stored[i+3]))

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
	remaining := float64((left0 + left1) + (left2 + left3))
	right := float64((right0 + right1) + (right2 + right3))

	for ; i < n; i++ {
		q, s := query[i], int32(int8(stored[i]))
		dot += float64(q * float32(s))
		remaining += float64(q * q)
		right += float64(s * s)
	}

	if remaining == 0 || right == 0 {
		return 0
	}
	return dot / math.Sqrt(remaining*right)
}
