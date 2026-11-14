package engine

import (
	"sync"

	"github.com/jiva-studio/numen/libs/shared/tokenizer"
)

// referenceLength is the token length at which a full batch of pairs is one pass.
const referenceLength = 512

// bucketStep is what a length past the smallest buckets is rounded up to a multiple of.
const bucketStep = 512

func getAttentionBudget(batchPairs int) int {
	return batchPairs * referenceLength * referenceLength
}

func isWithinBudget(rows, longest, limit, batchPairs int) bool {
	seq := roundUpToBucket(longest, limit)
	return rows*seq*seq <= getAttentionBudget(batchPairs)
}

func roundUpToBucket(seq, limit int) int {
	switch {
	case seq <= 64:
		return 64
	case seq <= 128:
		return 128
	case seq <= 256:
		return 256
	case seq <= bucketStep:
		return bucketStep
	}
	rounded := (seq + bucketStep - 1) / bucketStep * bucketStep
	if rounded > limit {
		return max(limit, seq)
	}
	return rounded
}

// padding holds the input tensors of a batch: the ids, the mask and the token types, rows of seq values each.
type padding struct {
	ids, mask, types []int64
}

func newPaddingPool() *sync.Pool {
	return &sync.Pool{New: func() any { return &padding{ids: nil, mask: nil, types: nil} }}
}

// padBatch lays the pairs of batch out as rows of one length, filled with pad and masked out past each pair.
func padBatch(paddings *sync.Pool, batch []tokenizer.Encoding, pad, limit int) (rows, seq int, in *padding) {
	rows = len(batch)
	longest := 0
	for _, one := range batch {
		longest = max(longest, len(one.IDs))
	}
	seq = roundUpToBucket(longest, limit)

	in, ok := paddings.Get().(*padding)
	if !ok || in == nil {
		in = &padding{ids: nil, mask: nil, types: nil}
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
		for i, id := range one.IDs {
			ids[i] = int64(id)
			in.mask[pos+i] = 1
		}
		for i, typeID := range one.TypeIDs {
			in.types[pos+i] = int64(typeID)
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
