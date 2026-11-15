package searchquery

import (
	"cmp"
	"slices"
)

// Merge joins the chunk results of one file whose ranges overlap or lie at most gapBytes apart, taken in
// order of start. A joined result covers the range of its parts, holds their chunks in order of start,
// takes the highest of each score of its parts, and the text, selector, place and line of the part with the
// highest rounded score (ties: the lower start).
func Merge(results []SearchResult, gapBytes int) []SearchResult {
	sorted := slices.Clone(results)
	slices.SortStableFunc(sorted, func(a, b SearchResult) int {
		return cmp.Or(cmp.Compare(a.Start, b.Start), cmp.Compare(a.Length, b.Length))
	})
	var merged, group []SearchResult
	groupEnd := 0
	for _, r := range sorted {
		if len(group) > 0 && r.Start > groupEnd+gapBytes {
			merged = append(merged, join(group))
			group = nil
		}
		if len(group) == 0 {
			groupEnd = r.Start + r.Length
		}
		group = append(group, r)
		groupEnd = max(groupEnd, r.Start+r.Length)
	}
	if len(group) > 0 {
		merged = append(merged, join(group))
	}
	return merged
}

func join(group []SearchResult) SearchResult {
	if len(group) == 1 {
		return group[0]
	}
	best := group[0]
	for _, part := range group[1:] {
		if compareScores(part.Score, best.Score) < 0 {
			best = part
		}
	}
	joined := best
	joined.Chunks = nil
	start, end := group[0].Start, 0
	for _, part := range group {
		joined.Chunks = append(joined.Chunks, part.Chunks...)
		start = min(start, part.Start)
		end = max(end, part.Start+part.Length)
		joined.Score = getHigher(joined.Score, part.Score)
		joined.MeaningScore = getHigher(joined.MeaningScore, part.MeaningScore)
		joined.WordScore = getHigher(joined.WordScore, part.WordScore)
	}
	slices.SortStableFunc(joined.Chunks, func(a, b ChunkRange) int { return cmp.Compare(a.Start, b.Start) })
	joined.Start, joined.Length = start, end-start
	return joined
}
