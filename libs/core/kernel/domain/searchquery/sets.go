package searchquery

import (
	"fmt"
	"slices"
	"strings"
)

// The messages of the problems of document functions that Normalize finds.
const (
	messageEmptyKey     = "the key of %s is empty"
	messageEmptyHeading = "the heading of hasHeading is empty"
	messageSets         = "a query has at most %d distinct property, hasProperty and hasHeading conditions"
)

func (n *normalizer) parseOperands(operands []Value) []Value {
	parsed := make([]Value, len(operands))
	for i, operand := range operands {
		parsed[i] = operand
		parsed[i].Position = Position{}
		if operand.Kind != ValueDate {
			continue
		}
		instant, err := ParseDate(operand.Text, n.zone)
		if err != nil {
			n.addProblem(operand.Position, ErrInvalidDate.Error())
			continue
		}
		parsed[i] = Value{Kind: ValueTime, Time: instant}
	}
	return parsed
}

func (n *normalizer) addSet(raw RawLeaf) int {
	set := SetCondition{Function: raw.Function, Key: raw.Key, Text: raw.Text, Comparison: raw.Comparison, Operands: n.parseOperands(raw.Operands)}
	switch {
	case raw.Function == FunctionHasHeading && strings.TrimSpace(raw.Text) == "":
		n.addProblem(raw.Position, messageEmptyHeading)
		return 0
	case raw.Function != FunctionHasHeading && strings.TrimSpace(raw.Key) == "":
		n.addProblem(raw.Position, fmt.Sprintf(messageEmptyKey, raw.Function))
		return 0
	}
	if index := slices.IndexFunc(n.sets, func(s SetCondition) bool { return isSameSet(s, set) }); index >= 0 {
		return index
	}
	if len(n.sets) == n.limits.MaxConditions {
		n.addProblem(raw.Position, fmt.Sprintf(messageSets, n.limits.MaxConditions))
		return 0
	}
	n.sets = append(n.sets, set)
	return len(n.sets) - 1
}

func isSameSet(a, b SetCondition) bool {
	return a.Function == b.Function && a.Key == b.Key && a.Text == b.Text && a.Comparison == b.Comparison &&
		slices.EqualFunc(a.Operands, b.Operands, isSameValue)
}
