package chunk

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"
)

// coarseCandidates is how many chunks the coarse pass keeps for each answer
// that leaves the meaning half.
//
// One bit per dimension orders by Hamming distance, and the full-precision
// vectors decide among what that order kept. How coarse that first order is, is
// a fact about this index and not about the search, so the number is here.
const coarseCandidates = 8

// scored is one candidate and how near the query it turned out to be.
type scored struct {
	chunk      int64
	similarity float64
}

// rerank orders candidates by their full-precision similarity to the query,
// nearest first, and keeps only what reaches the floor.
//
// A candidate the index holds no comparable vector for cannot be compared and
// is not an answer. A vector that is read and does not compare is a corrupt
// row, and says so.
func (q *Queries) rerank(ctx context.Context, recipe string, query []float32, candidates []int64, floor float64) ([]int64, error) {
	ids, err := json.Marshal(candidates)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, stmt.Get("rerank"), string(ids), recipe)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	similarity := make(map[int64]float64, len(candidates))
	for rows.Next() {
		var chunk int64
		var stored []byte
		if err := rows.Scan(&chunk, &stored); err != nil {
			return nil, err
		}
		if len(stored) != len(query) {
			return nil, fmt.Errorf("chunk %d holds %d dimensions where the query has %d",
				chunk, len(stored), len(query))
		}
		similarity[chunk] = embedding.Similarity(query, signed(stored))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	kept := make([]scored, 0, len(candidates))
	for _, chunk := range candidates {
		if s, held := similarity[chunk]; held && s >= floor {
			kept = append(kept, scored{chunk: chunk, similarity: s})
		}
	}
	// Two chunks of equal similarity keep the coarse pass's order between them.
	sort.SliceStable(kept, func(a, b int) bool { return kept[a].similarity > kept[b].similarity })

	out := make([]int64, 0, len(kept))
	for _, k := range kept {
		out = append(out, k.chunk)
	}
	return out, nil
}

// signed reads a stored vector as the dimensions it holds, one per byte.
func signed(stored []byte) []int8 {
	out := make([]int8, len(stored))
	for i, b := range stored {
		out[i] = int8(b)
	}
	return out
}
