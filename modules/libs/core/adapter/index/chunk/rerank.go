package chunk

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
)

// coarseCandidates is how many chunks the coarse pass keeps for each answer
// that leaves the meaning half.
//
// One bit per dimension orders by Hamming distance, and the full-precision
// vectors decide among what that order kept. How coarse that first order is, is
// a fact about this index and not about the search, so the number is here.
const coarseCandidates = 8

// candidate is one candidate and how near the query it turned out to be.
type candidate struct {
	chunk      int64
	similarity float64
	order      int
}

// rerank orders candidates by their full-precision similarity to the query,
// nearest first, and keeps only what reaches the floor.
//
// A candidate the index holds no comparable vector for cannot be compared and
// is not an answer. A vector that is read and does not compare is a corrupt
// row, and says so.
func (q *Queries) rerank(ctx context.Context, recipe string, query []float32, candidates []int64, of []domain.SourceKind, floor float64) ([]int64, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	ids := formatInt64s(candidates)
	wanted := formatKinds(of)
	rows, err := q.db.QueryContext(ctx, stmt.Get("rerank"), ids, recipe, wanted)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	kept := make([]candidate, 0, len(candidates))
	var stored []byte
	for rows.Next() {
		var chunk int64
		if err := rows.Scan(&chunk, &stored); err != nil {
			return nil, err
		}
		if len(stored) != len(query) {
			return nil, fmt.Errorf("chunk %d holds %d dimensions where the query has %d",
				chunk, len(stored), len(query))
		}
		sim := embedding.SimilarityBytes(query, stored)
		if sim >= floor {
			kept = append(kept, candidate{
				chunk:      chunk,
				similarity: sim,
				order:      len(kept),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Two chunks of equal similarity keep the coarse pass's order between them.
	slices.SortFunc(kept, func(a, b candidate) int {
		if c := cmp.Compare(b.similarity, a.similarity); c != 0 {
			return c
		}
		return cmp.Compare(a.order, b.order)
	})

	out := make([]int64, len(kept))
	for i, k := range kept {
		out[i] = k.chunk
	}
	return out, nil
}

func formatInt64s(nums []int64) string {
	if len(nums) == 0 {
		return "[]"
	}
	b := make([]byte, 0, len(nums)*12+2)
	b = append(b, '[')
	for i, n := range nums {
		if i > 0 {
			b = append(b, ',')
		}
		b = strconv.AppendInt(b, n, 10)
	}
	b = append(b, ']')
	return string(b)
}

func formatKinds(chosen []domain.SourceKind) string {
	if len(chosen) == 0 {
		return "[]"
	}
	size := 2
	for _, k := range chosen {
		size += len(k) + 3
	}
	b := make([]byte, 0, size)
	b = append(b, '[')
	for i, k := range chosen {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '"')
		b = append(b, k...)
		b = append(b, '"')
	}
	b = append(b, ']')
	return string(b)
}
