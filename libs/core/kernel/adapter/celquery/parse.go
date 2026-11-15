// Package celquery parses the text of a search query with cel-go into the raw tree of searchquery. It parses
// and type-checks; it creates no CEL program.
package celquery

import (
	"fmt"
	"unicode/utf8"

	"cel.dev/cel-go/common"
	"cel.dev/cel-go/common/ast"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

// Parser parses queries within the depth and size of its limits.
type Parser struct {
	maxDepth      int
	maxCodePoints int
}

// NewParser returns a parser that refuses a query nested deeper than limits.MaxDepth or longer than
// limits.MaxQueryBytes code points.
func NewParser(limits searchquery.Limits) Parser {
	return Parser{maxDepth: limits.MaxDepth, maxCodePoints: limits.MaxQueryBytes}
}

// ParseQuery parses and checks the text against the catalog and converts it into a raw tree. A query that
// is not valid returns a *searchquery.ParseError with every problem found.
func (p Parser) ParseQuery(text string, catalog searchquery.Catalog) (searchquery.RawNode, error) {
	environment, err := newEnvironment(catalog, p.maxDepth, p.maxCodePoints)
	if err != nil {
		return searchquery.RawNode{}, fmt.Errorf("declare the query language: %w", err)
	}
	parsed, issues := environment.Parse(text)
	if issues.Err() != nil {
		return searchquery.RawNode{}, newParseError(issues.Errors())
	}
	native := parsed.NativeRep()
	names := nameWalk{catalog: catalog, source: native.SourceInfo()}
	names.walk(native.Expr())
	if len(names.problems) > 0 {
		return searchquery.RawNode{}, &searchquery.ParseError{Problems: names.problems}
	}
	checked, issues := environment.Check(parsed)
	if issues.Err() != nil {
		return searchquery.RawNode{}, newParseError(issues.Errors())
	}
	c := converter{source: checked.NativeRep().SourceInfo()}
	root := c.convert(checked.NativeRep().Expr())
	if len(c.problems) > 0 {
		return searchquery.RawNode{}, &searchquery.ParseError{Problems: c.problems}
	}
	return root, nil
}

func newParseError(errs []*common.Error) *searchquery.ParseError {
	problems := make([]searchquery.Problem, len(errs))
	for i, err := range errs {
		problems[i] = searchquery.Problem{Position: getPosition(err.Location), Message: err.Message}
	}
	return &searchquery.ParseError{Problems: problems}
}

func getPosition(location common.Location) searchquery.Position {
	return searchquery.Position{Line: max(location.Line(), 1), Column: max(location.Column()+1, 1)}
}

func getExprPosition(source *ast.SourceInfo, expr ast.Expr) searchquery.Position {
	// CEL places a call at its opening parenthesis.
	position := getPosition(source.GetStartLocation(expr.ID()))
	if expr.Kind() != ast.CallKind {
		return position
	}
	name := expr.AsCall().FunctionName()
	if isOperator(name) {
		return position
	}
	position.Column = max(position.Column-utf8.RuneCountInString(name), 1)
	return position
}
