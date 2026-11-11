package engine

import (
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/onnx"
)

// referenceLength is the token length at which a full configured batch is one pass.
const referenceLength = 512

func getAttentionBudget(batchTexts int) int {
	return batchTexts * referenceLength * referenceLength
}

func meanPool(flat []float32, mask []int64, rows, seq, dimensions int, isNormalised bool) [][]float32 {
	out := make([][]float32, rows)
	backing := make([]float32, rows*dimensions)
	for row := range rows {
		vector := backing[row*dimensions : (row+1)*dimensions]
		tokenCount := 0
		rowOffset := row * seq
		for token := range seq {
			if mask[rowOffset+token] == 0 {
				continue
			}
			tokenCount++
			pos := (rowOffset + token) * dimensions
			addRow(vector, flat[pos:pos+dimensions])
		}
		// gomutants:disable-next-line INTEGER_INCREMENT reason="dividing by a token count of one leaves the vector unchanged"
		if tokenCount > 0 {
			k := float32(tokenCount)
			d := 0
			// gomutants:disable-next-line LOOP_CONDITION,INTEGER_INCREMENT,CONDITIONALS_BOUNDARY reason="the scalar tail loop divides any remaining elements identically"
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
		out[row] = finishVector(vector, isNormalised)
	}
	return out
}

func addRow(dst, src []float32) {
	n := min(len(dst), len(src))
	d := 0
	// gomutants:disable-next-line LOOP_CONDITION,INTEGER_INCREMENT,CONDITIONALS_BOUNDARY reason="the scalar tail loop adds any remaining elements identically"
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

// tokenOutput is the per-token output of a model for a batch: rows texts of seq tokens, each token a vector
// of dimensions values, and the mask of the tokens that are not padding.
type tokenOutput struct {
	flat                  []float32
	mask                  []int64
	rows, seq, dimensions int
}

func pool(kind string, out tokenOutput, isNormalised bool) ([][]float32, error) {
	switch kind {
	case model.PoolMean:
		return meanPool(out.flat, out.mask, out.rows, out.seq, out.dimensions, isNormalised), nil
	case model.PoolCLS:
		return headPool(out.flat, out.rows, out.seq, out.dimensions, isNormalised), nil
	case model.PoolLast:
		return lastPool(out.flat, out.mask, out.rows, out.seq, out.dimensions, isNormalised), nil
	}
	return nil, fmt.Errorf("a per-token output cannot be pooled as %q: the model answers one vector per text only as an output named sentence_embedding", kind)
}

func headPool(flat []float32, rows, seq, dimensions int, isNormalised bool) [][]float32 {
	out := make([][]float32, rows)
	for row := range out {
		pos := row * seq * dimensions
		vector := make([]float32, dimensions)
		copy(vector, flat[pos:pos+dimensions])
		out[row] = finishVector(vector, isNormalised)
	}
	return out
}

func lastPool(flat []float32, mask []int64, rows, seq, dimensions int, isNormalised bool) [][]float32 {
	out := make([][]float32, rows)
	for row := range out {
		last := 0
		for token := range seq {
			if mask[row*seq+token] != 0 {
				last = token
			}
		}
		pos := (row*seq + last) * dimensions
		vector := make([]float32, dimensions)
		copy(vector, flat[pos:pos+dimensions])
		out[row] = finishVector(vector, isNormalised)
	}
	return out
}

// padding holds the three input tensors of a batch.
type padding struct {
	ids, mask, types, positions []int64
}

func newPaddingPool() *sync.Pool {
	return &sync.Pool{New: func() any { return &padding{ids: nil, mask: nil, types: nil, positions: nil} }}
}

func padBatch(paddings *sync.Pool, batch [][]int, pad, limit int) (rows, seq int, in *padding) {
	rows = len(batch)
	seq = roundUpToBucket(getLongest(batch), limit)

	in, ok := paddings.Get().(*padding)
	if !ok || in == nil {
		in = &padding{ids: nil, mask: nil, types: nil, positions: nil}
	}
	size := rows * seq
	in.ids = resize(in.ids, size)
	in.mask = resize(in.mask, size)
	in.types = resize(in.types, size)
	in.positions = resize(in.positions, size)
	clear(in.mask)
	clear(in.types)
	clear(in.positions)
	for row, one := range batch {
		pos := row * seq
		ids := in.ids[pos : pos+seq]
		for i := range ids {
			ids[i] = int64(pad)
		}
		for i, id := range one {
			ids[i] = int64(id)
			in.mask[pos+i] = 1
			in.positions[pos+i] = int64(i)
		}
		if len(one) == 0 {
			in.mask[pos] = 1
		}
	}
	return rows, seq, in
}

func getLongest(batch [][]int) int {
	longest := 0
	for _, one := range batch {
		longest = max(longest, len(one))
	}
	return longest
}

// bucketStep is what a length past the smallest buckets is rounded up to a multiple of.
const bucketStep = 512

func roundUpToBucket(seq, limit int) int {
	switch {
	case seq <= 64:
		return 64
	case seq <= 128:
		return 128
	case seq <= 256:
		return 256
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="a length equal to bucketStep rounds to bucketStep through either branch"
	case seq <= bucketStep:
		return bucketStep
	}
	rounded := (seq + bucketStep - 1) / bucketStep * bucketStep
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="a rounded length equal to limit returns limit through either branch"
	if rounded > limit {
		return max(limit, seq)
	}
	return rounded
}

func resize(s []int64, size int) []int64 {
	if cap(s) < size {
		return make([]int64, size)
	}
	return s[:size]
}

func getWindow(provider onnx.Provider, maxTokens int) int {
	if provider == onnx.ProviderCoreML {
		return min(maxTokens, referenceLength)
	}
	return maxTokens
}
