package searchquery

// ConditionScore is how one chunk scores against one condition. Similarity is the best cosine of the small
// chunks inside the chunk with a meaning text; WordScore the best negated bm25 of a small chunk inside it for
// a words expression; IsMatch whether the whole chunk text matches a words expression.
type ConditionScore struct {
	Similarity *float64
	WordScore  *float64
	IsMatch    bool
}

// Outcome is whether a chunk matches the tree, and the scores that decided it.
type Outcome struct {
	IsMatch      bool
	MeaningScore *float64
	WordScore    *float64
}

// Evaluate evaluates the tree for one chunk of the document. A meaning leaf holds at or above the floor; a
// negated one fails when the chunk reaches the floor for its text and is closer to it than to every positive
// meaning condition. A words leaf holds when the whole chunk matches. && takes the lowest score of each
// family over its children, || the highest over its matching children.
func Evaluate(root Node, conditions []Condition, scores []ConditionScore, floor float64, d Document) Outcome {
	e := evaluator{scores: scores, floor: floor, document: d, positive: getBestPositiveSimilarity(conditions, scores)}
	return e.evaluate(root)
}

type evaluator struct {
	scores   []ConditionScore
	floor    float64
	document Document
	positive *float64
}

func getBestPositiveSimilarity(conditions []Condition, scores []ConditionScore) *float64 {
	var best *float64
	for i, c := range conditions {
		if !c.IsMeaning || !c.IsPositive || i >= len(scores) {
			continue
		}
		if s := scores[i].Similarity; s != nil && (best == nil || *s > *best) {
			best = s
		}
	}
	return best
}

func (e evaluator) evaluate(node Node) Outcome {
	switch node.Operator {
	case OperatorAnd:
		return e.evaluateAnd(node.Children)
	case OperatorOr:
		return e.evaluateOr(node.Children)
	case OperatorLeaf:
		return e.evaluateLeaf(node)
	case OperatorNot:
		return Outcome{}
	}
	return Outcome{}
}

func (e evaluator) evaluateAnd(children []Node) Outcome {
	result := Outcome{IsMatch: true}
	for _, child := range children {
		o := e.evaluate(child)
		if !o.IsMatch {
			return Outcome{}
		}
		result.MeaningScore = getLower(result.MeaningScore, o.MeaningScore)
		result.WordScore = getLower(result.WordScore, o.WordScore)
	}
	return result
}

func (e evaluator) evaluateOr(children []Node) Outcome {
	var result Outcome
	for _, child := range children {
		o := e.evaluate(child)
		if !o.IsMatch {
			continue
		}
		result.IsMatch = true
		result.MeaningScore = getHigher(result.MeaningScore, o.MeaningScore)
		result.WordScore = getHigher(result.WordScore, o.WordScore)
	}
	return result
}

func (e evaluator) evaluateLeaf(node Node) Outcome {
	leaf := node.Leaf
	switch leaf.Kind {
	case LeafMeaning:
		return e.evaluateMeaning(e.getScore(leaf.ConditionIndex), node.IsNegated)
	case LeafWords:
		return e.evaluateWords(e.getScore(leaf.ConditionIndex), node.IsNegated)
	case LeafKindIn, LeafPathPrefix:
		return Outcome{IsMatch: isDocumentLeafMatch(leaf, e.document) != node.IsNegated}
	}
	return Outcome{}
}

func (e evaluator) getScore(index int) ConditionScore {
	if index < 0 || index >= len(e.scores) {
		return ConditionScore{}
	}
	return e.scores[index]
}

func (e evaluator) evaluateMeaning(score ConditionScore, isNegated bool) Outcome {
	similarity := score.Similarity
	isReached := similarity != nil && *similarity >= e.floor
	if !isNegated {
		if !isReached {
			return Outcome{}
		}
		value := *similarity
		return Outcome{IsMatch: true, MeaningScore: &value}
	}
	if !isReached {
		return Outcome{IsMatch: true}
	}
	isCloser := e.positive == nil || *similarity > *e.positive
	return Outcome{IsMatch: !isCloser}
}

func (e evaluator) evaluateWords(score ConditionScore, isNegated bool) Outcome {
	if isNegated {
		return Outcome{IsMatch: !score.IsMatch}
	}
	if !score.IsMatch {
		return Outcome{}
	}
	var wordScore *float64
	if score.WordScore != nil {
		value := *score.WordScore
		wordScore = &value
	}
	return Outcome{IsMatch: true, WordScore: wordScore}
}

func getLower(a, b *float64) *float64 {
	if a == nil || (b != nil && *b < *a) {
		return b
	}
	return a
}

func getHigher(a, b *float64) *float64 {
	if a == nil || (b != nil && *b > *a) {
		return b
	}
	return a
}
