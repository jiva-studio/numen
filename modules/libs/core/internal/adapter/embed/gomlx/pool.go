package gomlx

import "github.com/jiva-studio/numen/modules/libs/core/internal/embedding"

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

// headPool turns a model's per-token output into one vector per text: the
// first token, at unit length.
//
// A model trained this way gathers what a text says into the token that opens
// it, and the tokens after it carry nothing a vector is made of.
func headPool(flat []float32, rows, seq, dimensions int) [][]float32 {
	out := make([][]float32, rows)
	for row := range out {
		at := row * seq * dimensions
		vector := make([]float32, dimensions)
		copy(vector, flat[at:at+dimensions])
		out[row] = embedding.Normalise(vector)
	}
	return out
}

// padded lays a batch out as the model takes it: one row per text, each row the
// same length, and the rows a full batch would carry.
//
// A pass is one shape, and a shape is compiled the first time it appears. The
// rows a batch does not fill carry the padding token, and each is marked at one
// token so that what pools a row divides by something.
func padBatch(batch [][]int, rows, seq, pad int) (ids, mask, types [][]int64) {
	ids = make([][]int64, rows)
	mask = make([][]int64, rows)
	types = make([][]int64, rows)
	for row := range rows {
		ids[row] = make([]int64, seq)
		mask[row] = make([]int64, seq)
		types[row] = make([]int64, seq)
		for i := range ids[row] {
			ids[row][i] = int64(pad)
		}
		if row >= len(batch) {
			mask[row][0] = 1
			continue
		}
		for i, id := range batch[row][:min(len(batch[row]), seq)] {
			ids[row][i] = int64(id)
			mask[row][i] = 1
		}
	}
	return ids, mask, types
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
