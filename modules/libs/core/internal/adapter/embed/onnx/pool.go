package onnx

import (
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
)

// meanPool turns a model's per-token output into one vector per text: the
// average of the tokens the mask keeps, at unit length.
//
// Padding is excluded, so a text's vector is the same however long the other
// texts in its batch are.
func meanPool(flat []float32, mask []int64, rows, seq, dimensions int) [][]float32 {
	out := make([][]float32, rows)
	backing := make([]float32, rows*dimensions)
	for row := range rows {
		vector := backing[row*dimensions : (row+1)*dimensions]
		kept := 0
		rowOffset := row * seq
		for token := 0; token < seq; token++ {
			if mask[rowOffset+token] == 0 {
				continue
			}
			kept++
			at := (rowOffset + token) * dimensions
			addRow(vector, flat[at:at+dimensions])
		}
		if kept > 0 {
			k := float32(kept)
			d := 0
			for ; d+8 <= dimensions; d += 8 {
				vector[d] /= k
				vector[d+1] /= k
				vector[d+2] /= k
				vector[d+3] /= k
				vector[d+4] /= k
				vector[d+5] /= k
				vector[d+6] /= k
				vector[d+7] /= k
			}
			for ; d < dimensions; d++ {
				vector[d] /= k
			}
		}
		out[row] = embedding.Normalise(vector)
	}
	return out
}

// addRow adds src to dst element by element, eight elements a step.
func addRow(dst, src []float32) {
	n := min(len(dst), len(src))
	d := 0
	for ; d+8 <= n; d += 8 {
		dst[d] += src[d]
		dst[d+1] += src[d+1]
		dst[d+2] += src[d+2]
		dst[d+3] += src[d+3]
		dst[d+4] += src[d+4]
		dst[d+5] += src[d+5]
		dst[d+6] += src[d+6]
		dst[d+7] += src[d+7]
	}
	for ; d < n; d++ {
		dst[d] += src[d]
	}
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

// padding holds the three input tensors of a batch.
type padding struct {
	ids, mask, types []int64
}

var paddings = sync.Pool{New: func() any { return &padding{} }}

// releasePadding hands a padding back for the next batch. Nothing reads its
// slices afterwards.
func releasePadding(p *padding) {
	paddings.Put(p)
}

// padBatch lays a batch out as the model takes it: one row per text, every row
// as long as the longest of them, the rest of a row the padding token.
//
// The three come back as the model reads them, row after row, in a padding the
// caller releases once the batch is done with.
func padBatch(batch [][]int, pad int) (rows, seq int, in *padding) {
	rows = len(batch)
	for _, one := range batch {
		seq = max(seq, len(one))
	}
	// A batch of empty texts is still a batch, and a model takes no sequence of
	// no tokens.
	seq = max(seq, 1)

	in, _ = paddings.Get().(*padding)
	if in == nil {
		in = &padding{}
	}
	size := rows * seq
	in.ids = resize(in.ids, size)
	in.mask = resize(in.mask, size)
	in.types = resize(in.types, size)
	clear(in.mask)
	clear(in.types)
	for row, one := range batch {
		at := row * seq
		ids := in.ids[at : at+seq]
		for i := range ids {
			ids[i] = int64(pad)
		}
		for i, id := range one {
			ids[i] = int64(id)
			in.mask[at+i] = 1
		}
		// A row the mask keeps nothing of is a row whose average divides by
		// nothing, and the padding is what the text amounts to.
		if len(one) == 0 {
			in.mask[at] = 1
		}
	}
	return rows, seq, in
}

func resize(s []int64, size int) []int64 {
	if cap(s) < size {
		return make([]int64, size)
	}
	return s[:size]
}
