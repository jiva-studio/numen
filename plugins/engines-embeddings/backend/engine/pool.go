package engine

import (
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

// referenceLength is the token length at which a full configured batch is one pass.
const referenceLength = 512

// getAttentionBudget bounds one forward pass by rows * seq^2.
func getAttentionBudget(batchTexts int) int {
	return batchTexts * referenceLength * referenceLength
}

// meanPool turns a model's per-token output into one vector per text: the
// average of the tokens the mask keeps, at unit length.
func meanPool(flat []float32, mask []int64, rows, seq, dimensions int) [][]float32 {
	out := make([][]float32, rows)
	backing := make([]float32, rows*dimensions)
	for row := range rows {
		vector := backing[row*dimensions : (row+1)*dimensions]
		tokenCount := 0
		rowOffset := row * seq
		for token := 0; token < seq; token++ {
			if mask[rowOffset+token] == 0 {
				continue
			}
			tokenCount++
			pos := (rowOffset + token) * dimensions
			addRow(vector, flat[pos:pos+dimensions])
		}
		if tokenCount > 0 {
			k := float32(tokenCount)
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
		out[row] = Normalise(vector)
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
func headPool(flat []float32, rows, seq, dimensions int) [][]float32 {
	out := make([][]float32, rows)
	for row := range out {
		pos := row * seq * dimensions
		vector := make([]float32, dimensions)
		copy(vector, flat[pos:pos+dimensions])
		out[row] = Normalise(vector)
	}
	return out
}

// padding holds the three input tensors of a batch.
type padding struct {
	ids, mask, types []int64
}

var paddings = sync.Pool{New: func() any { return &padding{} }}

// releasePadding hands a padding back for the next batch.
func releasePadding(p *padding) {
	paddings.Put(p)
}

// padBatch lays a batch out as the model takes it.
func padBatch(batch [][]int, pad, limit int) (rows, seq int, in *padding) {
	rows = len(batch)
	seq = roundUpToBucket(getLongest(batch), limit)

	var isPadding bool
	in, isPadding = paddings.Get().(*padding)
	if !isPadding {
		in = &padding{}
	}
	size := rows * seq
	in.ids = resize(in.ids, size)
	in.mask = resize(in.mask, size)
	in.types = resize(in.types, size)
	clear(in.mask)
	clear(in.types)
	for row, one := range batch {
		pos := row * seq
		ids := in.ids[pos : pos+seq]
		for i := range ids {
			ids[i] = int64(pad)
		}
		for i, id := range one {
			ids[i] = int64(id)
			in.mask[pos+i] = 1
		}
		if len(one) == 0 {
			in.mask[pos] = 1
		}
	}
	return rows, seq, in
}

// getLongest is the length of the longest row of a batch.
func getLongest(batch [][]int) int {
	longest := 0
	for _, one := range batch {
		longest = max(longest, len(one))
	}
	return longest
}

// bucketStep is what a length past the smallest buckets is rounded up to a multiple of.
const bucketStep = 512

// roundUpToBucket rounds a sequence length up to the next discrete boundary.
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

// getWindow is the most tokens a text keeps for a session on provider.
func getWindow(provider onnx.Provider, maxTokens int) int {
	if provider == onnx.ProviderCoreML {
		return min(maxTokens, referenceLength)
	}
	return maxTokens
}
