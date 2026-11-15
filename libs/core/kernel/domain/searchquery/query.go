package searchquery

// Operator is what a node of the tree does with its children.
type Operator int

const (
	// OperatorLeaf holds one leaf and no children.
	OperatorLeaf Operator = iota
	// OperatorAnd holds when every child holds.
	OperatorAnd
	// OperatorOr holds when any child holds.
	OperatorOr
	// OperatorNot holds when its one child does not; it appears only in a RawNode.
	OperatorNot
)

// LeafKind is what a leaf tests.
type LeafKind int

const (
	LeafMeaning LeafKind = iota
	LeafWords
	// LeafKindIn is kind == "x" or kind in ["x", …].
	LeafKindIn
	// LeafPathPrefix is path.startsWith("x").
	LeafPathPrefix
)

// Position is a place in the text of a query, counted from 1; Column counts Unicode code points.
type Position struct {
	Line, Column int
}

// Leaf is one test of a normalized tree. ConditionIndex is set for LeafMeaning and LeafWords.
type Leaf struct {
	Kind           LeafKind
	ConditionIndex int
	Kinds          []string
	Prefix         string
	Position       Position
}

// IsChunkLevel reports whether the leaf decides on one chunk.
func (l Leaf) IsChunkLevel() bool {
	return l.Kind == LeafMeaning || l.Kind == LeafWords
}

// Node is a node of a normalized tree. Negation stands only on leaves.
type Node struct {
	Operator  Operator
	Children  []Node
	Leaf      Leaf
	IsNegated bool
}

// Condition is a distinct meaning or words text. IsPositive is set when it appears at least once without
// a negation.
type Condition struct {
	IsMeaning  bool
	Text       string
	IsPositive bool
}

// Stage is one method call on the results of a query.
type Stage struct {
	Name     string
	Args     []string
	Position Position
}

// Query is a parsed and normalized query. IsSearchResults is set when the query names a chunk-level
// function; a query without one is a document predicate.
type Query struct {
	Root            Node
	Conditions      []Condition
	Stages          []Stage
	IsSearchResults bool
}

// RawLeaf is one test as a parser emits it.
type RawLeaf struct {
	Kind     LeafKind
	Text     string
	Kinds    []string
	Prefix   string
	Position Position
}

// RawNode is the tree a parser emits, with negations where they are written.
type RawNode struct {
	Operator Operator
	Children []RawNode
	Leaf     RawLeaf
}
