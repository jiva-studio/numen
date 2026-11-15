// Package ranking orders the results of a saved search and joins neighboring results of one file.
package ranking

import (
	"cmp"
	"math"
	"slices"
)

// Candidate is a passage a search found, or neighboring passages of one file joined. Start and Length
// locate it in the text of the file, in bytes; Text, Selector, PlaceLabel and MatchLine are those of the
// search result of its best passage. Score is the similarity by meaning or the reranking score, nil for a
// passage found by words, kind or path alone; WordScore the best score of the searches by words, nil when
// none found it; ChunkCount how many passages were joined into it, and ChunkIDs their chunk ids in the
// order of their start.
type Candidate struct {
	ChunkID       string
	Path          string
	Start, Length int
	Text          string
	Selector      string
	PlaceLabel    string
	MatchLine     int
	Score         *float64
	WordScore     *float64
	ChunkCount    int
	ChunkIDs      []string
}

const decimals = 1000

// Round rounds a score to 3 decimals, so two runs over one index order equal scores the same way.
func Round(score float64) float64 {
	return math.Round(score*decimals) / decimals
}

// Sort orders the candidates by their score rounded to 3 decimals, highest first, the candidates with a
// score before those without; those without by their word score, highest first; then by path and start.
func Sort(candidates []Candidate) {
	slices.SortStableFunc(candidates, Compare)
}

// Compare is the order of Sort.
func Compare(a, b Candidate) int {
	if order := compareScores(a.Score, b.Score, true); order != 0 {
		return order
	}
	if a.Score == nil {
		if order := compareScores(a.WordScore, b.WordScore, false); order != 0 {
			return order
		}
	}
	return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Start, b.Start), cmp.Compare(a.Length, b.Length))
}

func compareScores(a, b *float64, isRounded bool) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	}
	left, right := *a, *b
	if isRounded {
		left, right = Round(left), Round(right)
	}
	return cmp.Compare(right, left)
}
