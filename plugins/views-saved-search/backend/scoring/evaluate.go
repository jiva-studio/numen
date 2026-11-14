// children, || the highest score of its matching children. A path prefix is compared in Unicode normal form
// C, so a path in either form matches a prefix in either form.
// Package scoring evaluates a compiled query for one candidate from the scores the host gave its chunk.
package scoring

import (
	"slices"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/query"
)

// Scores are the scores of one chunk, one for each leaf of the query, in the order of the leaves.
type Scores []host.ConditionScore

// Outcome is whether a candidate matches the query, and its fused score: the similarity by meaning that
// decided the match; nil when it matched through words, kind or path alone.
type Outcome struct {
	IsMatch bool
	Score   *float64
}

// Evaluate evaluates the tree for the chunk with the scores at the file path. A meaning leaf holds at or
// above the floor. A negated meaning leaf drops the chunk only when it reaches the floor for its text and
// is closer to it than to every meaning leaf that is not negated. && takes the lowest score of its
// children, || the highest score of its matching children.
func Evaluate(root query.Node, leaves []query.Leaf, scores Scores, floor float64, filePath string) Outcome {
	e := evaluator{
		leaves: leaves, scores: scores, floor: floor, path: filePath,
		positive: getBestPositiveSimilarity(leaves, scores),
	}
	return e.evaluate(root)
}

type evaluator struct {
	leaves   []query.Leaf
	scores   Scores
	floor    float64
	path     string
	positive *float64
}

func getBestPositiveSimilarity(leaves []query.Leaf, scores Scores) *float64 {
	var best *float64
	for i, leaf := range leaves {
		if !leaf.IsMeaning || !leaf.IsPositive || i >= len(scores) {
			continue
		}
		if similarity := scores[i].Similarity; similarity != nil && (best == nil || *similarity > *best) {
			best = similarity
		}
	}
	return best
}

func (e evaluator) evaluate(node query.Node) Outcome {
	switch node.Operator {
	case query.OperatorAnd:
		return e.evaluateAnd(node.Children)
	case query.OperatorOr:
		return e.evaluateOr(node.Children)
	case query.OperatorLeaf:
		return e.evaluateLeaf(node)
	}
	return Outcome{IsMatch: false, Score: nil}
}

func (e evaluator) evaluateAnd(children []query.Node) Outcome {
	var score *float64
	for _, child := range children {
		outcome := e.evaluate(child)
		if !outcome.IsMatch {
			return Outcome{IsMatch: false, Score: nil}
		}
		if outcome.Score != nil && (score == nil || *outcome.Score < *score) {
			score = outcome.Score
		}
	}
	return Outcome{IsMatch: true, Score: score}
}

func (e evaluator) evaluateOr(children []query.Node) Outcome {
	result := Outcome{IsMatch: false, Score: nil}
	for _, child := range children {
		outcome := e.evaluate(child)
		if !outcome.IsMatch {
			continue
		}
		result.IsMatch = true
		if outcome.Score != nil && (result.Score == nil || *outcome.Score > *result.Score) {
			result.Score = outcome.Score
		}
	}
	return result
}

func (e evaluator) evaluateLeaf(node query.Node) Outcome {
	term := node.Term
	switch term.Kind {
	case query.TermMeaning:
		return e.evaluateMeaning(e.getScore(term.LeafIndex), node.IsNegated)
	case query.TermWords:
		return Outcome{IsMatch: e.getScore(term.LeafIndex).IsMatch != node.IsNegated, Score: nil}
	case query.TermKind:
		return Outcome{IsMatch: slices.Contains(term.Kinds, query.GetKind(e.path)) != node.IsNegated, Score: nil}
	case query.TermPath:
		return Outcome{IsMatch: hasPathPrefix(e.path, term.Prefix) != node.IsNegated, Score: nil}
	}
	return Outcome{IsMatch: false, Score: nil}
}

func (e evaluator) getScore(index int) host.ConditionScore {
	if index < 0 || index >= len(e.scores) {
		return host.ConditionScore{}
	}
	return e.scores[index]
}

func (e evaluator) evaluateMeaning(score host.ConditionScore, isNegated bool) Outcome {
	similarity := score.Similarity
	isReached := similarity != nil && *similarity >= e.floor
	if !isNegated {
		if !isReached {
			return Outcome{IsMatch: false, Score: nil}
		}
		value := *similarity
		return Outcome{IsMatch: true, Score: &value}
	}
	isCloser := e.positive == nil || (similarity != nil && *similarity > *e.positive)
	return Outcome{IsMatch: !(isReached && isCloser), Score: nil}
}

func hasPathPrefix(filePath, prefix string) bool {
	return strings.HasPrefix(norm.NFC.String(filePath), norm.NFC.String(prefix))
}
