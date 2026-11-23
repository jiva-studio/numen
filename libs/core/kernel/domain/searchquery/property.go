package searchquery

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// PropertyKind is the type of a frontmatter value as stored.
type PropertyKind int

const (
	PropertyNull PropertyKind = iota
	PropertyBool
	PropertyInt
	PropertyDouble
	PropertyString
	PropertyList
	PropertyMap
)

// PropertyValue is the value of one frontmatter key as stored.
type PropertyValue struct {
	Kind   PropertyKind
	Bool   bool
	Int    int64
	Double float64
	Text   string
	Items  []PropertyValue
}

// PropertyEntry is the value of one frontmatter key in the note at Path.
type PropertyEntry struct {
	Path  string
	Value PropertyValue
}

// HeadingEntry is one heading of the note at Path.
type HeadingEntry struct {
	Path string
	Text string
}

// IsPropertyMatch reports whether a stored value satisfies a comparison of property("k") with the operands.
// A string compared with a date is read by ParseDate in zone; no other value changes type, so "4" is not 4.
// A list is equal to nothing, holds a literal when an item equals it, and is in a list when an item is.
func IsPropertyMatch(v PropertyValue, c Comparison, operands []Value, zone *time.Location) bool {
	if c == ComparisonPresent {
		return true
	}
	if len(operands) == 0 {
		return false
	}
	switch c {
	case ComparisonContains:
		if v.Kind == PropertyList {
			return slices.ContainsFunc(v.Items, func(item PropertyValue) bool { return isPropertyEqual(item, operands[0], zone) })
		}
		return v.Kind != PropertyNull && isPropertyEqual(v, operands[0], zone)
	case ComparisonIn:
		items := []PropertyValue{v}
		if v.Kind == PropertyList {
			items = v.Items
		}
		return slices.ContainsFunc(items, func(item PropertyValue) bool {
			return slices.ContainsFunc(operands, func(operand Value) bool { return isPropertyEqual(item, operand, zone) })
		})
	case ComparisonEqual:
		return isPropertyEqual(v, operands[0], zone)
	default:
		order, isOrdered := getPropertyOrder(v, operands[0], zone)
		return isOrdered && isOrderMatch(order, c)
	}
}

func isPropertyEqual(v PropertyValue, operand Value, zone *time.Location) bool {
	switch v.Kind {
	case PropertyNull:
		return operand.Kind == ValueNull
	case PropertyBool:
		return operand.Kind == ValueBool && v.Bool == operand.Bool
	case PropertyInt, PropertyDouble, PropertyString:
		order, isOrdered := getPropertyOrder(v, operand, zone)
		return isOrdered && order == 0
	case PropertyList, PropertyMap:
		return false
	}
	return false
}

func getPropertyOrder(v PropertyValue, operand Value, zone *time.Location) (int, bool) {
	switch {
	case v.Kind == PropertyString && operand.Kind == ValueString:
		return strings.Compare(v.Text, operand.Text), true
	case v.Kind == PropertyString && operand.Kind == ValueTime:
		instant, err := ParseDate(v.Text, zone)
		if err != nil {
			return 0, false
		}
		return instant.Compare(operand.Time), true
	case v.Kind == PropertyInt && operand.Kind == ValueInt:
		return cmp.Compare(v.Int, operand.Int), true
	case isNumber(v) && (operand.Kind == ValueInt || operand.Kind == ValueDouble):
		return compareNumbers(getPropertyNumber(v), getValueNumber(operand)), true
	}
	return 0, false
}

func isNumber(v PropertyValue) bool {
	return v.Kind == PropertyInt || v.Kind == PropertyDouble
}

type number struct {
	isInt    bool
	whole    int64
	fraction float64
}

func getPropertyNumber(v PropertyValue) number {
	if v.Kind == PropertyInt {
		return number{isInt: true, whole: v.Int}
	}
	return number{fraction: v.Double}
}

func getValueNumber(v Value) number {
	if v.Kind == ValueInt {
		return number{isInt: true, whole: v.Int}
	}
	return number{fraction: v.Double}
}

func compareNumbers(a, b number) int {
	switch {
	case a.isInt && b.isInt:
		return cmp.Compare(a.whole, b.whole)
	case a.isInt:
		return compareWholeToFraction(a.whole, b.fraction)
	case b.isInt:
		return -compareWholeToFraction(b.whole, a.fraction)
	}
	return cmp.Compare(a.fraction, b.fraction)
}

func compareWholeToFraction(whole int64, fraction float64) int {
	switch {
	case math.IsNaN(fraction):
		return 1
	case fraction >= maxInt64AsFloat:
		return -1
	case fraction < -maxInt64AsFloat:
		return 1
	}
	truncated := math.Trunc(fraction)
	if order := cmp.Compare(whole, int64(truncated)); order != 0 {
		return order
	}
	return cmp.Compare(0, fraction-truncated)
}

const maxInt64AsFloat = float64(1 << 63)

// IsHeadingMatch reports whether a heading of a note is the heading text: the whole of it, without the spaces
// around it, ignoring case.
func IsHeadingMatch(heading, text string) bool {
	return domain.NormalizeName(strings.TrimSpace(heading)) == domain.NormalizeName(strings.TrimSpace(text))
}
