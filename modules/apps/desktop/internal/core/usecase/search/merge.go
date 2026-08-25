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

// where is one place in one file. A hit and the chunk enclosing it are two
// rows standing in the same place, and one place is one passage.
type where struct {
	source        string
	start, length int
}

// collapse keeps at most `each` passages of one source, in the order it was
// given, so the best-ranked chunks of a source are the ones that survive.
//
// Where a document has a section named what was asked for, that section is the
// first passage it answers with. Which documents answer is asked of every
// ranking; where in one to stand is what its names say.
func collapse(fused, named []domain.Passage, each, limit int) []domain.Passage {
	sections := map[string]domain.Passage{}
	for _, p := range named {
		if _, held := sections[p.Source]; !held {
			sections[p.Source] = p
		}
	}

	out := make([]domain.Passage, 0, min(limit, len(fused)))
	taken := map[string]int{}
	opened := map[string]bool{}
	held := map[where]bool{}

	keep := func(p domain.Passage) bool {
		at := where{p.Source, p.Start, p.Length}
		if held[at] || taken[p.Source] >= each {
			return true
		}
		held[at] = true
		taken[p.Source]++
		out = append(out, p)
		return len(out) < limit
	}

	for _, p := range fused {
		if section, has := sections[p.Source]; has && !opened[p.Source] {
			opened[p.Source] = true
			if !keep(section) {
				break
			}
		}
		if !keep(p) {
			break
		}
	}
	return out
}
