package searchquery

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// Comparison is what a document leaf or a set condition tests against its operands.
type Comparison int

const (
	// ComparisonEqual is ==; != is a negated ==.
	ComparisonEqual Comparison = iota
	ComparisonLess
	ComparisonLessOrEqual
	ComparisonGreater
	ComparisonGreaterOrEqual
	// ComparisonIn is value in [literals].
	ComparisonIn
	// ComparisonContains is "x" in property("k").
	ComparisonContains
	ComparisonStartsWith
	ComparisonEndsWith
	// ComparisonSubstring is the contains method of a string.
	ComparisonSubstring
	// ComparisonPresent is hasProperty and hasHeading, which have no operand.
	ComparisonPresent
)

// Attribute is a column of a document that a LeafCompare reads.
type Attribute int

const (
	AttributeTitle Attribute = iota
	AttributeNoteType
	AttributeModified
	AttributeSize
)

// ValueKind is the type of a literal of a query.
type ValueKind int

const (
	ValueString ValueKind = iota
	ValueInt
	ValueDouble
	ValueBool
	ValueNull
	// ValueDate is the text of date("…") as a parser emits it; Normalize turns it into a ValueTime.
	ValueDate
	ValueTime
)

// Value is a literal of a query. Position is set by a parser for a ValueDate, for its problems.
type Value struct {
	Kind     ValueKind
	Text     string
	Int      int64
	Double   float64
	Bool     bool
	Time     time.Time
	Position Position
}

func isCompareMatch(leaf Leaf, d Document) bool {
	if len(leaf.Operands) == 0 {
		return false
	}
	operand := leaf.Operands[0]
	switch leaf.Attribute {
	case AttributeTitle:
		return isTextMatch(d.Title, leaf.Comparison, leaf.Operands)
	case AttributeNoteType:
		return isTextMatch(d.NoteType, leaf.Comparison, leaf.Operands)
	case AttributeModified:
		return isOrderMatch(d.Modified.Compare(operand.Time), leaf.Comparison)
	case AttributeSize:
		return isOrderMatch(cmp.Compare(d.Size, operand.Int), leaf.Comparison)
	}
	return false
}

func isTextMatch(text string, c Comparison, operands []Value) bool {
	operand := operands[0].Text
	switch c {
	case ComparisonEqual:
		return text == operand
	case ComparisonIn:
		return slices.ContainsFunc(operands, func(v Value) bool { return v.Text == text })
	case ComparisonStartsWith:
		return strings.HasPrefix(text, operand)
	case ComparisonEndsWith:
		return strings.HasSuffix(text, operand)
	case ComparisonSubstring:
		return strings.Contains(text, operand)
	default:
		return false
	}
}

func isOrderMatch(order int, c Comparison) bool {
	switch c {
	case ComparisonEqual:
		return order == 0
	case ComparisonLess:
		return order < 0
	case ComparisonLessOrEqual:
		return order <= 0
	case ComparisonGreater:
		return order > 0
	case ComparisonGreaterOrEqual:
		return order >= 0
	default:
		return false
	}
}

func isSameValue(a, b Value) bool {
	return a.Kind == b.Kind && a.Text == b.Text && a.Int == b.Int && a.Double == b.Double && a.Bool == b.Bool && a.Time.Equal(b.Time)
}
