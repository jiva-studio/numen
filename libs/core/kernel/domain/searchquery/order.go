package searchquery

import (
	"cmp"
	"math"
	"slices"
)

const scoreDecimals = 1000

// Round rounds a score to 3 decimals.
func Round(score float64) float64 {
	return math.Round(score*scoreDecimals) / scoreDecimals
}

// Sort orders the results: score rounded to 3 decimals, highest first, results without a score last; then
// path; then start.
func Sort(r *SearchResults) {
	slices.SortStableFunc(r.Results, func(a, b SearchResult) int {
		return cmp.Or(
			compareScores(a.Score, b.Score),
			cmp.Compare(a.Document.Path, b.Document.Path),
			cmp.Compare(a.Start, b.Start),
		)
	})
}

func compareScores(a, b *float64) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	}
	return cmp.Compare(Round(*b), Round(*a))
}
