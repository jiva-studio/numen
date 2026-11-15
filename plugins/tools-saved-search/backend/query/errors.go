package query

import (
	"fmt"
	"strings"
)

// CompileError is a query that does not compile: every problem with its place in the query text.
type CompileError struct {
	Problems []Problem
}

// Problem is one problem of a query. Line and Column count from 1; Column counts characters.
type Problem struct {
	Line    int
	Column  int
	Message string
}

func (e *CompileError) Error() string {
	parts := make([]string, len(e.Problems))
	for i, problem := range e.Problems {
		parts[i] = fmt.Sprintf("%d:%d: %s", problem.Line, problem.Column, problem.Message)
	}
	return "the query is not valid: " + strings.Join(parts, "; ")
}
