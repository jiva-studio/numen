// Package query compiles the query language of a saved search, CEL over meaning("…"), words("…"), kind and
// path, into a tree the plugin evaluates itself. CEL parses and type-checks the text; no CEL program runs.
package query

// Operator is what a node of the tree does with its children.
type Operator int

const (
	// OperatorLeaf is a node that holds one term and no children.
	OperatorLeaf Operator = iota
	// OperatorAnd holds when every child holds.
	OperatorAnd
	// OperatorOr holds when any child holds.
	OperatorOr
)

// TermType is what a term tests.
type TermType int

const (
	// TermMeaning is meaning("…"): the similarity of a chunk with a text reaches the floor of the vault.
	TermMeaning TermType = iota
	// TermWords is words("…"): the whole text of a chunk matches a full-text query.
	TermWords
	// TermKind is a comparison of kind with kinds of files.
	TermKind
	// TermPath is path.startsWith("…").
	TermPath
)

// Query is a compiled query. The index of a leaf in Leaves, the distinct meaning and words conditions in
// order of first appearance, is the index of its condition in a ScoreChunks call. Filter narrows the
// searches the host runs for the query.
type Query struct {
	Root   Node
	Leaves []Leaf
	Filter Filter
}

// Node is a node of the tree. Term and IsNegated are set for OperatorLeaf only: negations stand only on
// leaves.
type Node struct {
	Operator  Operator
	Children  []Node
	Term      Term
	IsNegated bool
}

// Term is the test of a leaf. LeafIndex is the index into Query.Leaves of a TermMeaning or TermWords, Kinds
// the kinds of files that satisfy a TermKind, Prefix the folder of a TermPath. Line and Column are where CEL
// places the term in the text of the query, counted from 1; Column counts characters.
type Term struct {
	Kind         TermType
	LeafIndex    int
	Kinds        []string
	Prefix       string
	Line, Column int
}

// Leaf is a meaning or words condition. Text is sent to the host as written. IsPositive is set when the
// leaf appears at least once without a negation; such a leaf fetches candidates.
type Leaf struct {
	IsMeaning  bool
	Text       string
	IsPositive bool
}
