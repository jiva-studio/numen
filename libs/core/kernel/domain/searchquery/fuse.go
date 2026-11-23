package searchquery

import (
	"cmp"
	"slices"
)

// FusionK is the constant of reciprocal rank fusion.
const FusionK = 60

// SetScores sets the score of every result from its meaning and word scores. ScoreMeaning takes the meaning
// score and ScoreWords the word score. ScoreFused ranks the results by each family, best first, nil last,
// ties by path then start, and takes (1/(k+rM) + 1/(k+rW)) / (2/(k+1)), where a nil family adds 0.
func SetScores(results []SearchResult, kind ScoreKind) {
	switch kind {
	case ScoreMeaning:
		for i := range results {
			results[i].Score = results[i].MeaningScore
		}
	case ScoreWords:
		for i := range results {
			results[i].Score = results[i].WordScore
		}
	case ScoreFused:
		setFusedScores(results)
	case ScoreNone:
		for i := range results {
			results[i].Score = nil
		}
	case ScoreRerank:
	}
}

func setFusedScores(results []SearchResult) {
	sums := make([]float64, len(results))
	addRanks(results, sums, func(r SearchResult) *float64 { return r.MeaningScore })
	addRanks(results, sums, func(r SearchResult) *float64 { return r.WordScore })
	best := 2.0 / float64(FusionK+1)
	for i := range results {
		score := sums[i] / best
		results[i].Score = &score
	}
}

func addRanks(results []SearchResult, sums []float64, get func(SearchResult) *float64) {
	order := make([]int, 0, len(results))
	for i, r := range results {
		if get(r) != nil {
			order = append(order, i)
		}
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Or(
			cmp.Compare(*get(results[b]), *get(results[a])),
			cmp.Compare(results[a].Document.Path, results[b].Document.Path),
			cmp.Compare(results[a].Start, results[b].Start),
		)
	})
	for rank, i := range order {
		sums[i] += 1.0 / float64(FusionK+rank+1)
	}
}
