package searchquery

import (
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Document is one source of a vault as the planner and a result see it. ChunkCount is the count of its small
// chunks that carry a vector under the recipe in use.
type Document struct {
	Path         string
	Title        string
	SourceKind   domain.SourceKind
	Kind         string
	ChunkCount   int
	ModifiedAt   time.Time
	Size         int64
	MatchedCalls []bool
}

// IsDocumentMatch evaluates the tree for the document with no chunk: a chunk leaf holds when it is negated.
func IsDocumentMatch(root Node, d Document) bool {
	switch root.Operator {
	case OperatorAnd:
		for _, child := range root.Children {
			if !IsDocumentMatch(child, d) {
				return false
			}
		}
		return true
	case OperatorOr:
		return slices.ContainsFunc(root.Children, func(child Node) bool { return IsDocumentMatch(child, d) })
	case OperatorLeaf:
		if root.Leaf.IsChunkLevel() {
			return root.IsNegated
		}
		return isDocumentLeafMatch(root.Leaf, d) != root.IsNegated
	case OperatorNot:
		return false
	}
	return false
}

func isDocumentLeafMatch(leaf Leaf, d Document) bool {
	switch leaf.Kind {
	case LeafFunction:
		return leaf.CallIndex < len(d.MatchedCalls) && d.MatchedCalls[leaf.CallIndex]
	case LeafKindIn:
		return slices.Contains(leaf.Kinds, d.Kind)
	case LeafPathPrefix:
		return domain.HasPathPrefix(d.Path, leaf.Prefix)
	case LeafMeaning, LeafWords:
		return false
	}
	return false
}
