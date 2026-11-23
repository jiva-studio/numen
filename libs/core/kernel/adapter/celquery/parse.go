// Package celquery parses the text of a search query with cel-go into the raw tree of searchquery. It parses
// and type-checks; it creates no CEL program.
package celquery

import (
	"cmp"
	"fmt"
	"slices"

	"cel.dev/cel-go/common"
	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/parser"

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
	source := newQueryText(text)
	parsed, issues := environment.Parse(text)
	if issues.Err() != nil {
		return searchquery.RawNode{}, p.newSyntaxError(source, catalog)
	}
	native := parsed.NativeRep()
	names := nameWalk{catalog: catalog, source: native.SourceInfo(), text: source}
	names.walk(native.Expr())
	if len(names.problems) > 0 {
		return searchquery.RawNode{}, &searchquery.ParseError{Problems: names.problems}
	}
	checked, issues := environment.Check(parsed)
	if issues.Err() != nil {
		return searchquery.RawNode{}, &searchquery.ParseError{Problems: newCheckProblems(source, issues.Errors(), catalog)}
	}
	s := &sink{source: checked.NativeRep().SourceInfo(), text: source}
	c := converter{sink: s, catalog: catalog, comparer: comparer{s}, links: linkConverter{s}}
	root := convertQuery(&c, checked.NativeRep().Expr())
	if len(c.problems) > 0 {
		return searchquery.RawNode{}, &searchquery.ParseError{Problems: c.problems}
	}
	return root, nil
}

func (p Parser) newSyntaxError(source queryText, catalog searchquery.Catalog) *searchquery.ParseError {
	recovering, err := parser.NewParser(
		parser.Macros(), parser.MaxRecursionDepth(p.maxDepth), parser.ExpressionSizeCodePointLimit(p.maxCodePoints),
	)
	if err != nil {
		return &searchquery.ParseError{Problems: []searchquery.Problem{{Position: searchquery.StartPosition(), Message: messageUnreadable}}}
	}
	partial, errs := recovering.Parse(common.NewTextSource(string(source.runes)))
	problems, quoted := newSyntaxProblems(source, errs.GetErrors())
	if partial != nil {
		names := nameWalk{catalog: catalog, source: partial.SourceInfo(), text: source}
		names.walk(partial.Expr())
		for _, problem := range names.problems {
			if !isInsideAny(source, quoted, problem.Position) {
				problems = append(problems, problem)
			}
		}
	}
	slices.SortStableFunc(problems, func(a, b searchquery.Problem) int {
		return cmp.Or(cmp.Compare(a.Position.Line, b.Position.Line), cmp.Compare(a.Position.Column, b.Position.Column))
	})
	return &searchquery.ParseError{Problems: problems}
}

func isInsideAny(source queryText, spans []span, position searchquery.Position) bool {
	offset := source.lineStarts[position.Line-1] + position.Column - 1
	return slices.ContainsFunc(spans, func(s span) bool { return s.hasOffset(offset) })
}

func getExprPosition(source queryText, info *ast.SourceInfo, expr ast.Expr) searchquery.Position {
	offset := source.getOffset(info.GetStartLocation(expr.ID()))
	if expr.Kind() == ast.CallKind && !isOperator(expr.AsCall().FunctionName()) {
		offset = source.findNameStart(offset)
	}
	return source.getPosition(offset)
}
