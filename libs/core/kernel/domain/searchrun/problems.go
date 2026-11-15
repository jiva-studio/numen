package searchrun

import (
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

// ReadProblems returns the problems of an error of a query: each problem of a *searchquery.ParseError with
// its position, any other error as one problem without position.
func ReadProblems(err error) []Problem {
	var parseErr *searchquery.ParseError
	if !errors.As(err, &parseErr) {
		return []Problem{{Message: err.Error()}}
	}
	problems := make([]Problem, len(parseErr.Problems))
	for i, p := range parseErr.Problems {
		problems[i] = Problem{Line: p.Position.Line, Column: p.Position.Column, Message: p.Message, Suggestion: p.Suggestion}
	}
	return problems
}
