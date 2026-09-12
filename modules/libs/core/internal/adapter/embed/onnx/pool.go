package onnx

import "github.com/jiva-studio/numen/modules/libs/core/internal/embedding"

// meanPool turns a model's per-token output into one vector per text: the
// average of the tokens the mask keeps, at unit length.
//
// Padding is excluded, so a text's vector is the same however long the other
// texts in its batch are.
func meanPool(flat []float32, mask []int64, rows, seq, dimensions int) [][]float32 {
	out := make([][]float32, rows)
	for row := range rows {
		vector := make([]float32, dimensions)
		kept := 0
		for token := 0; token < seq; token++ {
			if mask[row*seq+token] == 0 {
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

// padBatch lays a batch out as the model takes it: one row per text, every row
// as long as the longest of them, the rest of a row the padding token.
//
// The three come back as the model reads them, row after row.
func padBatch(batch [][]int, pad int) (rows, seq int, ids, mask, types []int64) {
	rows = len(batch)
	for _, one := range batch {
		seq = max(seq, len(one))
	}
	// A batch of empty texts is still a batch, and a model takes no sequence of
	// no tokens.
	seq = max(seq, 1)

	ids = make([]int64, rows*seq)
	mask = make([]int64, rows*seq)
	types = make([]int64, rows*seq)
	for row, one := range batch {
		at := row * seq
		for i := range seq {
			ids[at+i] = int64(pad)
		}
		for i, id := range one {
			ids[at+i] = int64(id)
			mask[at+i] = 1
		}
		// A row the mask keeps nothing of is a row whose average divides by
		// nothing, and the padding is what the text amounts to.
		if len(one) == 0 {
			mask[at] = 1
		}
	}
	return rows, seq, ids, mask, types
}
