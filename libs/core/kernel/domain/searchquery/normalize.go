package searchquery

import (
	"fmt"
	"slices"
	"strings"
)

// The messages of the problems Normalize and CheckQueryText find.
const (
	messageQueryBytes   = "a query has at most %d bytes"
	messageTextSize     = "%s needs a text of 1 to %d bytes"
	messageConditions   = "a query has at most %d distinct meaning and words conditions"
	messagePathPrefixes = "a query has at most %d distinct path prefixes"
	messageNoPositive   = "a query needs a condition that is not negated"
	messageKind         = "kind is one of %s"
)

// CheckQueryText returns a *ParseError when the text of a query is longer than the limit.
func CheckQueryText(text string, limits Limits) error {
	if len(text) <= limits.MaxQueryBytes {
		return nil
	}
	return &ParseError{Problems: []Problem{{Position: StartPosition(), Message: fmt.Sprintf(messageQueryBytes, limits.MaxQueryBytes)}}}
}

// Normalize turns the tree a parser emitted into a Query: negations pushed to the leaves by De Morgan,
// nested groups of one operator flattened, and equal conditions given one index. A tree that breaks a rule
// of the language returns a *ParseError with every problem.
func Normalize(raw RawNode, limits Limits) (Query, error) {
	n := normalizer{limits: limits}
	root := n.convert(raw, false)
	n.checkHasPositiveLeaf(root)
	q := Query{Root: root, Conditions: n.conditions, IsSearchResults: hasChunkLeaf(root)}
	q.Stages = n.convertStages(raw.Stages, GetDefaultRerankQuery(q))
	if len(n.problems) > 0 {
		return Query{}, &ParseError{Problems: n.problems}
	}
	return q, nil
}

type normalizer struct {
	limits     Limits
	conditions []Condition
	prefixes   []string
	problems   []Problem
}

func (n *normalizer) addProblem(position Position, message string) {
	n.problems = append(n.problems, Problem{Position: position, Message: message})
}

func (n *normalizer) convert(raw RawNode, isNegated bool) Node {
	switch raw.Operator {
	case OperatorNot:
		if len(raw.Children) != 1 {
			return Node{Operator: OperatorAnd}
		}
		return n.convert(raw.Children[0], !isNegated)
	case OperatorAnd, OperatorOr:
		return n.convertGroup(raw, isNegated)
	case OperatorLeaf:
		return n.convertLeaf(raw.Leaf, isNegated)
	}
	return Node{Operator: OperatorAnd}
}

func (n *normalizer) convertGroup(raw RawNode, isNegated bool) Node {
	operator := raw.Operator
	if isNegated {
		operator = getOtherOperator(operator)
	}
	group := Node{Operator: operator}
	for _, child := range raw.Children {
		converted := n.convert(child, isNegated)
		if converted.Operator == operator {
			group.Children = append(group.Children, converted.Children...)
			continue
		}
		group.Children = append(group.Children, converted)
	}
	return group
}

func getOtherOperator(operator Operator) Operator {
	if operator == OperatorAnd {
		return OperatorOr
	}
	return OperatorAnd
}

func (n *normalizer) convertLeaf(raw RawLeaf, isNegated bool) Node {
	leaf := Leaf{Kind: raw.Kind, Kinds: raw.Kinds, Prefix: raw.Prefix, Position: raw.Position}
	switch raw.Kind {
	case LeafMeaning, LeafWords:
		leaf.ConditionIndex = n.addCondition(raw, !isNegated)
	case LeafKindIn:
		n.checkKinds(raw)
	case LeafPathPrefix:
		n.addPrefix(raw)
	}
	return Node{Operator: OperatorLeaf, Leaf: leaf, IsNegated: isNegated}
}

func (n *normalizer) addCondition(raw RawLeaf, isPositive bool) int {
	isMeaning := raw.Kind == LeafMeaning
	if text := strings.TrimSpace(raw.Text); text == "" || len(text) > n.limits.MaxConditionBytes {
		name := FunctionWords
		if isMeaning {
			name = FunctionMeaning
		}
		n.addProblem(raw.Position, fmt.Sprintf(messageTextSize, name, n.limits.MaxConditionBytes))
		return 0
	}
	for i, c := range n.conditions {
		if c.IsMeaning == isMeaning && c.Text == raw.Text {
			n.conditions[i].IsPositive = c.IsPositive || isPositive
			return i
		}
	}
	if len(n.conditions) == n.limits.MaxConditions {
		n.addProblem(raw.Position, fmt.Sprintf(messageConditions, n.limits.MaxConditions))
		return 0
	}
	n.conditions = append(n.conditions, Condition{IsMeaning: isMeaning, Text: raw.Text, IsPositive: isPositive})
	return len(n.conditions) - 1
}

func (n *normalizer) checkKinds(raw RawLeaf) {
	for _, kind := range raw.Kinds {
		if !IsKind(kind) {
			n.addProblem(raw.Position, fmt.Sprintf(messageKind, strings.Join(quoteAll(ListKinds()), ", ")))
			return
		}
	}
}

func quoteAll(texts []string) []string {
	out := make([]string, len(texts))
	for i, text := range texts {
		out[i] = fmt.Sprintf("%q", text)
	}
	return out
}

func (n *normalizer) addPrefix(raw RawLeaf) {
	if slices.Contains(n.prefixes, raw.Prefix) {
		return
	}
	if len(n.prefixes) == n.limits.MaxPathPrefixes {
		n.addProblem(raw.Position, fmt.Sprintf(messagePathPrefixes, n.limits.MaxPathPrefixes))
		return
	}
	n.prefixes = append(n.prefixes, raw.Prefix)
}

func (n *normalizer) checkHasPositiveLeaf(root Node) {
	if len(n.problems) == 0 && !hasPositiveLeaf(root) {
		n.addProblem(StartPosition(), messageNoPositive)
	}
}

func hasPositiveLeaf(node Node) bool {
	if node.Operator == OperatorLeaf {
		return !node.IsNegated
	}
	return slices.ContainsFunc(node.Children, hasPositiveLeaf)
}

func hasChunkLeaf(node Node) bool {
	if node.Operator == OperatorLeaf {
		return node.Leaf.IsChunkLevel()
	}
	return slices.ContainsFunc(node.Children, hasChunkLeaf)
}

func isDocumentOnly(node Node) bool {
	if node.Operator == OperatorLeaf {
		return !node.Leaf.IsChunkLevel()
	}
	for _, child := range node.Children {
		if !isDocumentOnly(child) {
			return false
		}
	}
	return true
}
