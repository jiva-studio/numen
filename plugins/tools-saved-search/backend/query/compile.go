package query

import (
	"fmt"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// The messages of the problems Compile finds after CEL checked the query.
const (
	messageCondition    = "the query is a condition over meaning, words, kind and path"
	messageKind         = `kind is compared with "note", "pdf", "epub", "audio", "video" or "image"`
	messagePath         = `path is used as path.startsWith("folder/")`
	messageNoPositive   = "a query needs a meaning or words condition that is not negated"
	messageLiteralText  = "%s takes a string literal"
	messageTextSize     = "%s needs a text of 1 to %d bytes"
	messageLeafCount    = "a query has at most %d meaning and words conditions"
	firstLine           = 1
	firstColumn         = 1
	celColumnToOneBased = 1
)

// Compile parses and checks the text of a query and converts it into a tree. A query that does not compile
// returns a *CompileError with every problem found.
func Compile(text string) (Query, error) {
	environment, err := newEnvironment()
	if err != nil {
		return Query{}, fmt.Errorf("declare the query language: %w", err)
	}
	checked, issues := environment.Compile(text)
	if err := issues.Err(); err != nil {
		return Query{}, newCompileError(issues.Errors())
	}
	native := checked.NativeRep()
	c := &converter{source: native.SourceInfo(), leaves: nil, problems: nil}
	if !checked.OutputType().IsExactType(cel.BoolType) {
		return Query{}, &CompileError{Problems: []Problem{c.getProblem(native.Expr().ID(), messageCondition)}}
	}
	root := c.convert(native.Expr(), false)
	if len(c.problems) == 0 && !hasPositiveLeaf(c.leaves) {
		c.problems = append(c.problems, Problem{Line: firstLine, Column: firstColumn, Message: messageNoPositive})
	}
	if len(c.problems) > 0 {
		return Query{}, &CompileError{Problems: c.problems}
	}
	return Query{Root: root, Leaves: c.leaves, Filter: getFilter(root)}, nil
}

func newCompileError(errs []*common.Error) *CompileError {
	problems := make([]Problem, len(errs))
	for i, err := range errs {
		problems[i] = Problem{
			Line:    max(err.Location.Line(), firstLine),
			Column:  max(err.Location.Column()+celColumnToOneBased, firstColumn),
			Message: err.Message,
		}
	}
	return &CompileError{Problems: problems}
}

func hasPositiveLeaf(leaves []Leaf) bool {
	for _, leaf := range leaves {
		if leaf.IsPositive {
			return true
		}
	}
	return false
}

func getLeafCountMessage() string {
	return fmt.Sprintf(messageLeafCount, host.MaxScoredConditions)
}
