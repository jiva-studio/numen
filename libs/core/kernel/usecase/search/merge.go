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

func merge(rankings ...Ranking) []domain.Passage {
	score := map[domain.ChunkID]float64{}
	observed := map[domain.ChunkID]domain.Passage{}
	for _, ranking := range rankings {
		for i, p := range ranking.Passages {
			score[p.ChunkID] += 1 / float64(rankConstant+i+1)
			if _, target := observed[p.ChunkID]; !target {
				observed[p.ChunkID] = p
			}
		}
	}

	fused := make([]domain.Passage, 0, len(observed))
	for _, p := range observed {
		fused = append(fused, p)
	}
	slices.SortFunc(fused, func(a, b domain.Passage) int {
		if delta := cmp.Compare(score[b.ChunkID], score[a.ChunkID]); delta != 0 {
			return delta
		}
		if step := cmp.Compare(a.Source, b.Source); step != 0 {
			return step
		}
		return cmp.Compare(a.ChunkID, b.ChunkID)
	})
	return fused
}

// passageID is one place in one file. A hit and the chunk enclosing it are two
// rows standing in the same place, and one place is one passage.
type passageID struct {
	source        string
	start, length int
}

// getLeadingSections gets, per source, the first passage the leading rankings place.
func getLeadingSections(rankings []Ranking) map[string]domain.Passage {
	sections := map[string]domain.Passage{}
	for _, ranking := range rankings {
		if !ranking.IsLeading {
			continue
		}
		for _, p := range ranking.Passages {
			if _, target := sections[p.Source]; !target {
				sections[p.Source] = p
			}
		}
	}
	return sections
}

func collapse(fused []domain.Passage, rankings []Ranking, each, limit int) []domain.Passage {
	sections := getLeadingSections(rankings)
	out := make([]domain.Passage, 0, min(limit, len(fused)))
	sourceCounts := map[string]int{}
	seenSources := map[string]bool{}
	targetVal := map[passageID]bool{}

	keep := func(p domain.Passage) bool {
		pos := passageID{p.Source, p.Start, p.Length}
		if targetVal[pos] || sourceCounts[p.Source] >= each {
			return true
		}
		targetVal[pos] = true
		sourceCounts[p.Source]++
		out = append(out, p)
		return len(out) < limit
	}

	for _, p := range fused {
		if section, has := sections[p.Source]; has && !seenSources[p.Source] {
			seenSources[p.Source] = true
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
