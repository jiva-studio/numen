package search

import (
	"cmp"
	"slices"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// rankConstant flattens the head of a ranking: first place is worth 1/61 and
// second 1/62, so a chunk both halves placed well outscores one that either half
// placed first alone.
//
// 60 is the constant reciprocal rank fusion uses.
const rankConstant = 60

// merge fuses rankings into one order.
//
// A chunk's score is the sum of `1 / (rankConstant + rank)` over the rankings
// that returned it, counting from one. BM25 and cosine similarity are not
// comparable — one is unbounded and drawn from the corpus, the other a bounded
// angle — so ranks are what is combined, and combining ranks calibrates nothing.
//
// Equal scores are ordered by source and then by chunk, so one index gives one
// answer.
func merge(rankings ...[]domain.Passage) []domain.Passage {
	score := map[int64]float64{}
	seen := map[int64]domain.Passage{}
	for _, ranking := range rankings {
		for i, p := range ranking {
			score[p.Chunk] += 1 / float64(rankConstant+i+1)
			if _, held := seen[p.Chunk]; !held {
				seen[p.Chunk] = p
			}
		}
	}

	fused := make([]domain.Passage, 0, len(seen))
	for _, p := range seen {
		fused = append(fused, p)
	}
	slices.SortFunc(fused, func(a, b domain.Passage) int {
		if by := cmp.Compare(score[b.Chunk], score[a.Chunk]); by != 0 {
			return by
		}
		if by := cmp.Compare(a.Source, b.Source); by != 0 {
			return by
		}
		return cmp.Compare(a.Chunk, b.Chunk)
	})
	return fused
}

// collapse keeps one passage per source: several passages of one document are
// one result, and a document names itself once.
//
// The order is the order it was given, so the best-ranked chunk of a source is
// the one that survives.
func collapse(fused []domain.Passage, limit int) []domain.Passage {
	out := make([]domain.Passage, 0, min(limit, len(fused)))
	taken := map[string]bool{}
	for _, p := range fused {
		if taken[p.Source] {
			continue
		}
		taken[p.Source] = true
		out = append(out, p)
		if len(out) == limit {
			break
		}
	}
	return out
}
