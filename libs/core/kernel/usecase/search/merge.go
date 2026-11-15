package search

import (
	"cmp"
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// rankConstant flattens the head of a ranking: first place is worth 1/61 and
// second 1/62, so a chunk two participants placed well outscores one that either
// placed first alone.
//
// 60 is the constant reciprocal rank fusion uses.
const rankConstant = 60

func merge(rankings ...Ranking) []domain.SearchResult {
	score := map[domain.ChunkID]float64{}
	observed := map[domain.ChunkID]domain.SearchResult{}
	for _, ranking := range rankings {
		for i, p := range ranking.Results {
			score[p.ChunkID] += 1 / float64(rankConstant+i+1)
			if _, target := observed[p.ChunkID]; !target {
				observed[p.ChunkID] = p
			}
		}
	}

	fused := make([]domain.SearchResult, 0, len(observed))
	for _, p := range observed {
		fused = append(fused, p)
	}
	slices.SortFunc(fused, func(a, b domain.SearchResult) int {
		if delta := cmp.Compare(score[b.ChunkID], score[a.ChunkID]); delta != 0 {
			return delta
		}
		if step := cmp.Compare(a.Path, b.Path); step != 0 {
			return step
		}
		return cmp.Compare(a.ChunkID, b.ChunkID)
	})
	return fused
}

// resultPlace is one place in one file. A hit and the chunk enclosing it are two
// rows standing in the same place, and one place is one search result.
type resultPlace struct {
	source        string
	start, length int
}

func getLeadingSections(rankings []Ranking) map[string]domain.SearchResult {
	sections := map[string]domain.SearchResult{}
	for _, ranking := range rankings {
		if !ranking.IsLeading {
			continue
		}
		for _, p := range ranking.Results {
			if _, target := sections[p.Path]; !target {
				sections[p.Path] = p
			}
		}
	}
	return sections
}

func collapse(fused []domain.SearchResult, rankings []Ranking, each, limit int) []domain.SearchResult {
	sections := getLeadingSections(rankings)
	out := make([]domain.SearchResult, 0, min(limit, len(fused)))
	sourceCounts := map[string]int{}
	seenSources := map[string]bool{}
	targetVal := map[resultPlace]bool{}

	keep := func(p domain.SearchResult) bool {
		pos := resultPlace{p.Path, p.Start, p.Length}
		if targetVal[pos] || sourceCounts[p.Path] >= each {
			return true
		}
		targetVal[pos] = true
		sourceCounts[p.Path]++
		out = append(out, p)
		return len(out) < limit
	}

	for _, p := range fused {
		if section, has := sections[p.Path]; has && !seenSources[p.Path] {
			seenSources[p.Path] = true
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
