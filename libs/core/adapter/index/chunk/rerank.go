package chunk

import (
	"cmp"
	"context"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
	"slices"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/internal/vector"
	"github.com/jiva-studio/numen/libs/notes/domain"
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
// nearest first, and keeps only those at or above the floor.
//
// A candidate with no comparable vector in the index cannot be compared and is
// not a result. A vector that is read and does not compare is a corrupt row,
// and is reported as an error.
func (q *Queries) rerank(ctx context.Context, recipe string, query []float32, candidates []int64, of []domain.SourceKind, floor float64) ([]int64, error) {
	ranked, err := q.rankCandidates(ctx, recipe, query, candidates, of, floor)
	if err != nil {
		return nil, err
	}
	if ranked == nil {
		return nil, nil
	}
	out := make([]int64, len(ranked))
	for i, k := range ranked {
		out[i] = k.chunk
	}
	return out, nil
}

// rankCandidates is rerank with the similarity of each candidate kept.
func (q *Queries) rankCandidates(ctx context.Context, recipe string, query []float32, candidates []int64, of []domain.SourceKind, floor float64) (_ []candidate, err error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	ids := formatInt64s(candidates)
	wanted := formatKinds(of)
	rows, err := q.db.QueryContext(ctx, stmt.Get("rerank"), ids, recipe, wanted)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	keptCandidates := make([]candidate, 0, len(candidates))
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
		sim := vector.SimilarityBytes(query, stored)
		if sim >= floor {
			keptCandidates = append(keptCandidates, candidate{
				chunk:      chunk,
				similarity: sim,
				order:      len(keptCandidates),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Two chunks of equal similarity keep the coarse pass's order between them.
	slices.SortFunc(keptCandidates, func(a, b candidate) int {
		if c := cmp.Compare(b.similarity, a.similarity); c != 0 {
			return c
		}
		return cmp.Compare(a.order, b.order)
	})

	return keptCandidates, nil
}

func formatInt64s(nums []int64) string {
	if len(nums) == 0 {
		return "[]"
	}
	// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated values are 13 and 3: a larger capacity hint; the text is the same at any capacity"
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

func formatKinds(selected []domain.SourceKind) string {
	if len(selected) == 0 {
		return "[]"
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="the mutated values are 1 and 3: the text needs 1 byte less than the hint, so a hint one smaller or larger yields the same text without growing"
	size := 2
	for _, k := range selected {
		// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is 4: a larger capacity hint; the text is the same at any capacity"
		size += len(k) + 3
	}
	b := make([]byte, 0, size)
	b = append(b, '[')
	for i, k := range selected {
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
