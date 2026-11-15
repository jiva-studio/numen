package searchquery

import (
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Scope is the document predicate every candidate of one condition satisfies, and its superset for the
// index: path prefixes (the whole vault when empty), source kinds (all when empty) and, when IsWithinPaths is
// set, the paths of documents a plugin function found. A Predicate that is an OperatorAnd with no children
// holds for every document.
type Scope struct {
	Predicate     Node
	PathPrefixes  []string
	SourceKinds   []domain.SourceKind
	Paths         []string
	IsWithinPaths bool
}

// IsEveryDocument reports whether the scope holds for every document.
func (s Scope) IsEveryDocument() bool {
	return s.Predicate.Operator == OperatorAnd && len(s.Predicate.Children) == 0
}

func (s Scope) IsDocumentMatch(d Document) bool {
	return IsDocumentMatch(s.Predicate, d)
}

// GetScopes returns the scope of each condition of the query, in the order of the conditions. The scope of
// one occurrence of a positive condition is the conjunction of the document-only children of every && on its
// path to the root; a condition that occurs in several places takes the disjunction of its scopes. A
// condition that is never positive has the scope of every document. Found are the calls whose every path is
// known, which bound the superset.
func GetScopes(q Query, found FoundPaths) []Scope {
	occurrences := make([][][]Node, len(q.Conditions))
	collectOccurrences(q.Root, nil, occurrences)
	scopes := make([]Scope, len(q.Conditions))
	for i, list := range occurrences {
		scopes[i] = newScope(list, found)
	}
	return scopes
}

// GetDocumentScope returns the scope of the documents the query may match as whole files: the predicate is
// the whole tree, the superset comes from its document-only conjuncts.
func GetDocumentScope(q Query, found FoundPaths) Scope {
	var conjuncts []Node
	switch {
	case isDocumentOnly(q.Root):
		conjuncts = []Node{q.Root}
	case q.Root.Operator == OperatorAnd:
		for _, child := range q.Root.Children {
			if isDocumentOnly(child) {
				conjuncts = append(conjuncts, child)
			}
		}
	}
	s := Scope{Predicate: q.Root, PathPrefixes: getPrefixes(conjuncts), SourceKinds: getSourceKinds(conjuncts)}
	s.Paths, s.IsWithinPaths = getFoundPaths(conjuncts, found)
	return s
}

func collectOccurrences(node Node, constraints []Node, occurrences [][][]Node) {
	switch node.Operator {
	case OperatorLeaf:
		if node.Leaf.IsChunkLevel() && !node.IsNegated && node.Leaf.ConditionIndex < len(occurrences) {
			index := node.Leaf.ConditionIndex
			occurrences[index] = append(occurrences[index], slices.Clone(constraints))
		}
	case OperatorAnd:
		collectAndOccurrences(node, constraints, occurrences)
	case OperatorOr:
		for _, child := range node.Children {
			collectOccurrences(child, constraints, occurrences)
		}
	case OperatorNot:
	}
}

func collectAndOccurrences(node Node, constraints []Node, occurrences [][][]Node) {
	inner := slices.Clone(constraints)
	for _, child := range node.Children {
		if isDocumentOnly(child) {
			inner = append(inner, child)
		}
	}
	for _, child := range node.Children {
		if !isDocumentOnly(child) {
			collectOccurrences(child, inner, occurrences)
		}
	}
}

func newScope(occurrences [][]Node, found FoundPaths) Scope {
	every := Scope{Predicate: Node{Operator: OperatorAnd}}
	if len(occurrences) == 0 || slices.ContainsFunc(occurrences, func(c []Node) bool { return len(c) == 0 }) {
		return every
	}
	if len(occurrences) == 1 {
		c := occurrences[0]
		s := Scope{Predicate: Node{Operator: OperatorAnd, Children: c}, PathPrefixes: getPrefixes(c), SourceKinds: getSourceKinds(c)}
		s.Paths, s.IsWithinPaths = getFoundPaths(c, found)
		return s
	}
	scope := Scope{Predicate: Node{Operator: OperatorOr}}
	prefixes, kinds := []string{}, []domain.SourceKind{}
	isWithinPaths := true
	var paths []string
	for _, c := range occurrences {
		scope.Predicate.Children = append(scope.Predicate.Children, Node{Operator: OperatorAnd, Children: c})
		prefixes = joinSuperset(prefixes, getPrefixes(c))
		kinds = joinSuperset(kinds, getSourceKinds(c))
		one, isFound := getFoundPaths(c, found)
		isWithinPaths = isWithinPaths && isFound
		paths = append(paths, one...)
	}
	if isWithinPaths {
		slices.Sort(paths)
		scope.Paths, scope.IsWithinPaths = slices.Compact(paths), true
	}
	scope.PathPrefixes, scope.SourceKinds = prefixes, kinds
	return scope
}

func joinSuperset[T comparable](all, one []T) []T {
	if all == nil || len(one) == 0 {
		return nil
	}
	for _, value := range one {
		if !slices.Contains(all, value) {
			all = append(all, value)
		}
	}
	return all
}

func getPrefixes(conjuncts []Node) []string {
	for _, c := range conjuncts {
		if prefixes := getPrefixesOf(c); len(prefixes) > 0 {
			return prefixes
		}
	}
	return nil
}

func getPrefixesOf(node Node) []string {
	if isPositivePrefix(node) {
		return []string{node.Leaf.Prefix}
	}
	if node.Operator != OperatorOr || !slices.ContainsFunc(node.Children, isPositivePrefix) {
		return nil
	}
	var prefixes []string
	for _, child := range node.Children {
		if !isPositivePrefix(child) {
			return nil
		}
		if !slices.Contains(prefixes, child.Leaf.Prefix) {
			prefixes = append(prefixes, child.Leaf.Prefix)
		}
	}
	return prefixes
}

func isPositivePrefix(node Node) bool {
	return node.Operator == OperatorLeaf && node.Leaf.Kind == LeafPathPrefix && !node.IsNegated
}

func getSourceKinds(conjuncts []Node) []domain.SourceKind {
	for _, c := range conjuncts {
		if c.Operator != OperatorLeaf || c.Leaf.Kind != LeafKindIn || c.IsNegated || len(c.Leaf.Kinds) == 0 {
			continue
		}
		first := GetSourceKind(c.Leaf.Kinds[0])
		if !slices.ContainsFunc(c.Leaf.Kinds, func(k string) bool { return GetSourceKind(k) != first }) {
			return []domain.SourceKind{first}
		}
	}
	return nil
}

func getFoundPaths(conjuncts []Node, found FoundPaths) ([]string, bool) {
	var paths []string
	isWithinPaths := false
	for _, c := range conjuncts {
		if c.Operator != OperatorLeaf || c.Leaf.Kind != LeafFunction || c.IsNegated {
			continue
		}
		one, isFound := found[c.Leaf.CallIndex]
		if !isFound {
			continue
		}
		if !isWithinPaths {
			paths, isWithinPaths = slices.Clone(one), true
			continue
		}
		kept := make(map[string]bool, len(one))
		for _, p := range one {
			kept[p] = true
		}
		paths = slices.DeleteFunc(paths, func(p string) bool { return !kept[p] })
	}
	return paths, isWithinPaths
}
