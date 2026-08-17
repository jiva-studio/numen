package onnx

import "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"

// meanPool turns a model's per-token output into one vector per text: the
// average of the tokens the mask keeps, at unit length.
//
// Padding is excluded, so a text's vector is the same however long the other
// texts in its batch are.
func meanPool(flat []float32, mask [][]int64, dimensions int) [][]float32 {
	out := make([][]float32, len(mask))
	seq := 0
	if len(mask) > 0 {
		seq = len(mask[0])
	}
	for row := range mask {
		vector := make([]float32, dimensions)
		kept := 0
		for token := 0; token < seq; token++ {
			if mask[row][token] == 0 {
				continue
			}
			kept++
			at := (row*seq + token) * dimensions
			for d := 0; d < dimensions; d++ {
				vector[d] += flat[at+d]
			}
		}
		if kept > 0 {
			for d := range vector {
				vector[d] /= float32(kept)
			}
		}
		out[row] = embedding.Normalise(vector)
	}
	return out
}

// bucket rounds a sequence length up to the next step. Every distinct shape
// costs a compilation, so the lengths are held to a few.
func bucket(tokens, step, limit int) int {
	if tokens > limit {
		tokens = limit
	}
	rounded := (tokens + step - 1) / step * step
	if rounded < step {
		rounded = step
	}
	if rounded > limit {
		rounded = limit
	}
	return rounded
}
