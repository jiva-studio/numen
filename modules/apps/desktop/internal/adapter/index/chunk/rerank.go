package chunk

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"
)

// coarseCandidates is how many chunks the coarse pass keeps for each answer
// that leaves the meaning half. One bit per dimension orders by Hamming
// distance, and the full-precision vectors decide among what that order kept.
const coarseCandidates = 8

// Floor is the cosine similarity a chunk reaches to be an answer. A
// nearest-neighbour query answers with k rows whatever was asked, and this is
// what leaves a query the vault has nothing for with none of them.
const Floor = 0.50

// scored is one candidate and how near the query it turned out to be.
type scored struct {
	chunk      int64
	similarity float64
}

// rerank orders candidates by their full-precision similarity to the query,
// nearest first, and keeps only what reaches the floor.
//
// A candidate the index holds no full-precision vector for cannot be compared
// and is not an answer.
func (q *Queries) rerank(ctx context.Context, query []float32, candidates []int64) ([]int64, error) {
	ids, err := json.Marshal(candidates)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, stmt.Get("rerank"), string(ids))
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
		similarity[chunk] = embedding.Similarity(query, signed(stored))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	kept := make([]scored, 0, len(candidates))
	for _, chunk := range candidates {
		if s, held := similarity[chunk]; held && s >= Floor {
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
