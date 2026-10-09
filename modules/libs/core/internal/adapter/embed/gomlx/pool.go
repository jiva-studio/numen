package gomlx

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
	for row := range rows {
		vector := make([]float32, dimensions)
		kept := 0
		for token := 0; token < seq; token++ {
			if mask[row*seq+token] == 0 {
				continue
			}
			kept++
			at := (row*seq + token) * dimensions
			addRow(vector, flat[at:at+dimensions])
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

// addRow adds src to dst element by element, four elements a step. Each
// element sums its tokens in order.
func addRow(dst, src []float32) {
	src = src[:len(dst)]
	d := 0
	for ; d+4 <= len(dst); d += 4 {
		dst[d] += src[d]
		dst[d+1] += src[d+1]
		dst[d+2] += src[d+2]
		dst[d+3] += src[d+3]
	}
	for ; d < len(dst); d++ {
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

// padBatch lays a batch out as the model takes it: one row per text, each row
// the same length, and the rows a full batch would carry.
//
// A pass is one shape, and a shape is compiled the first time it appears. The
// rows a batch does not fill carry the padding token, and each is marked at one
// token so that what pools a row divides by something.
func padBatch(batch [][]int, rows, seq, pad int) *padding {
	in, _ := paddings.Get().(*padding)
	if in == nil {
		in = &padding{}
	}
	size := rows * seq
	in.ids = resize(in.ids, size)
	in.mask = resize(in.mask, size)
	in.types = resize(in.types, size)
	clear(in.mask)
	clear(in.types)
	for row := range rows {
		at := row * seq
		ids := in.ids[at : at+seq]
		for i := range ids {
			ids[i] = int64(pad)
		}
		if row >= len(batch) {
			in.mask[at] = 1
			continue
		}
		for i, id := range batch[row][:min(len(batch[row]), seq)] {
			ids[i] = int64(id)
			in.mask[at+i] = 1
		}
		if len(batch[row]) == 0 {
			in.mask[at] = 1
		}
	}
	return in
}

func resize(s []int64, size int) []int64 {
	if cap(s) < size {
		return make([]int64, size)
	}
	return s[:size]
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
