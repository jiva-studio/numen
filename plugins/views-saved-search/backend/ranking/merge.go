package ranking

import (
	"cmp"
	"slices"
)

// Merge joins the candidates of one file whose ranges overlap or lie at most gap bytes apart, one after
// another. A joined candidate covers the range of all its parts, takes the highest score and word score
// of its parts, and the text, selector and line of the part that Compare puts first. The result is in no
// particular order.
func Merge(candidates []Candidate, gap int) []Candidate {
	byPath := map[string][]Candidate{}
	var paths []string
	for _, candidate := range candidates {
		if _, found := byPath[candidate.Path]; !found {
			paths = append(paths, candidate.Path)
		}
		byPath[candidate.Path] = append(byPath[candidate.Path], candidate)
	}
	var merged []Candidate
	for _, filePath := range paths {
		merged = append(merged, mergeFile(byPath[filePath], gap)...)
	}
	return merged
}

func mergeFile(candidates []Candidate, gap int) []Candidate {
	slices.SortStableFunc(candidates, func(a, b Candidate) int {
		return cmp.Or(cmp.Compare(a.Start, b.Start), cmp.Compare(a.Length, b.Length))
	})
	var merged []Candidate
	var group []Candidate
	groupEnd := 0
	for _, candidate := range candidates {
		if len(group) > 0 && candidate.Start > groupEnd+gap {
			merged = append(merged, join(group))
			group = nil
		}
		if len(group) == 0 {
			groupEnd = candidate.Start + candidate.Length
		}
		group = append(group, candidate)
		groupEnd = max(groupEnd, candidate.Start+candidate.Length)
	}
	if len(group) > 0 {
		merged = append(merged, join(group))
	}
	return merged
}

func join(group []Candidate) Candidate {
	if len(group) == 1 {
		return group[0]
	}
	best := slices.MinFunc(group, Compare)
	start, end, count := group[0].Start, 0, 0
	var score, wordScore *float64
	for _, part := range group {
		start = min(start, part.Start)
		end = max(end, part.Start+part.Length)
		count += part.ChunkCount
		score = getHigher(score, part.Score)
		wordScore = getHigher(wordScore, part.WordScore)
	}
	best.Start, best.Length = start, end-start
	best.Score, best.WordScore, best.ChunkCount = score, wordScore, count
	return best
}

func getHigher(a, b *float64) *float64 {
	if a == nil || (b != nil && *b > *a) {
		return b
	}
	return a
}
