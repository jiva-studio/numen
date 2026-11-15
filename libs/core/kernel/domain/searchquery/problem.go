package searchquery

import (
	"fmt"
	"strings"
)

// Problem is one problem of a query. End is the last character it covers when it covers more than one,
// else zero. Suggestion is a catalog name close to a name the query misspelled.
type Problem struct {
	Position   Position
	End        Position
	Message    string
	Suggestion string
}

// ParseError is a query that is not valid, with every problem found.
type ParseError struct {
	Problems []Problem
}

func (e *ParseError) Error() string {
	parts := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		parts[i] = FormatProblem(p)
	}
	return "the query is not valid: " + strings.Join(parts, "; ")
}

// FormatProblem returns "line:column: message", with ": did you mean name?" when there is a suggestion.
func FormatProblem(p Problem) string {
	text := fmt.Sprintf("%d:%d: %s", p.Position.Line, p.Position.Column, p.Message)
	if p.Suggestion != "" {
		text += fmt.Sprintf(": did you mean %s?", p.Suggestion)
	}
	return text
}

// StartPosition is the first character of a query.
func StartPosition() Position {
	return Position{Line: 1, Column: 1}
}
