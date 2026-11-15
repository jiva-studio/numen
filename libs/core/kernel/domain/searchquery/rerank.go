package searchquery

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

// ErrNoReranker is a rerank stage in an installation whose Reranking setting has no row.
var ErrNoReranker = errors.New("no reranking service is set; add one under Settings, System, Reranking")

// RerankFailure is one row of the Reranking setting that did not answer.
type RerankFailure struct {
	ConnectionName string
	Reason         string
}

// RerankError is a rerank stage that every row of the Reranking setting failed.
type RerankError struct {
	Failures []RerankFailure
}

func (e *RerankError) Error() string {
	parts := make([]string, len(e.Failures))
	for i, f := range e.Failures {
		parts[i] = f.ConnectionName + ": " + f.Reason
	}
	return "no reranking service answered: " + strings.Join(parts, "; ")
}

// RerankReport says which row answered one rerank stage, how many chunks were sent to it and how many
// scores were reused from earlier runs.
type RerankReport struct {
	ConnectionID   string
	ConnectionName string
	Options        map[string]string
	SentCount      int
	ReusedCount    int
}

// RerankUnit is one text a rerank stage scores: a chunk of a result, or a whole-file result when
// ChunkIndex is -1.
type RerankUnit struct {
	ResultIndex int
	ChunkIndex  int
}

// GetDefaultRerankQuery joins the texts of the positive meaning and words conditions with a newline, in
// condition order.
func GetDefaultRerankQuery(q Query) string {
	var texts []string
	for _, c := range q.Conditions {
		if c.IsPositive {
			texts = append(texts, c.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// TruncateText returns the first most bytes of text, cut back to a rune boundary.
func TruncateText(text string, most int) string {
	if len(text) <= most {
		return text
	}
	end := most
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end]
}

// SelectRerankUnits returns the units to send: the chunks of the results whose documents are among the first
// window distinct documents, in result order and inside a result in order of start, at most maxReranked. A
// whole-file result is one unit.
func SelectRerankUnits(r SearchResults, window, maxReranked int) []RerankUnit {
	inWindow := getWindowPaths(r, window)
	var units []RerankUnit
	for i, result := range r.Results {
		if !inWindow[result.Document.Path] {
			continue
		}
		if result.IsWholeFile() {
			units = append(units, RerankUnit{ResultIndex: i, ChunkIndex: -1})
		}
		for _, j := range getChunkOrder(result.Chunks) {
			units = append(units, RerankUnit{ResultIndex: i, ChunkIndex: j})
		}
		if len(units) >= maxReranked {
			return units[:maxReranked]
		}
	}
	return units
}

func getWindowPaths(r SearchResults, window int) map[string]bool {
	paths := map[string]bool{}
	for _, result := range r.Results {
		if len(paths) == window {
			break
		}
		paths[result.Document.Path] = true
	}
	return paths
}

func getChunkOrder(chunks []ChunkRange) []int {
	order := make([]int, len(chunks))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(chunks[a].Start, chunks[b].Start) })
	return order
}

// ApplyRerank keeps the results of the first window documents that had a unit sent, gives each the highest
// score of its units, sets the kind of the scores to ScoreRerank and sorts. scores[i] belongs to units[i].
func ApplyRerank(r SearchResults, window int, units []RerankUnit, scores []float64) SearchResults {
	inWindow := getWindowPaths(r, window)
	best := map[int]float64{}
	for i, unit := range units {
		if score, found := best[unit.ResultIndex]; !found || scores[i] > score {
			best[unit.ResultIndex] = scores[i]
		}
	}
	var kept []SearchResult
	for i, result := range r.Results {
		score, found := best[i]
		if !found || !inWindow[result.Document.Path] {
			continue
		}
		result.Score, result.RerankScore = new(score), new(score)
		kept = append(kept, result)
	}
	r.Results = kept
	r.ScoreKind = ScoreRerank
	Sort(&r)
	return r
}
