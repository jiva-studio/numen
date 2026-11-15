package celquery

import (
	"slices"

	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	"cel.dev/cel-go/common/overloads"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

// nameWalk finds every function, stage and variable of a parsed query that the catalog does not declare.
type nameWalk struct {
	catalog  searchquery.Catalog
	source   *ast.SourceInfo
	text     queryText
	problems []searchquery.Problem
}

func (w *nameWalk) addProblem(expr ast.Expr, message, name string) {
	suggestion, _ := searchquery.SuggestName(name, w.catalog)
	w.problems = append(w.problems, searchquery.Problem{
		Position: getExprPosition(w.text, w.source, expr), Message: message + " " + name, Suggestion: suggestion,
	})
}

func (w *nameWalk) addMemberProblem(expr ast.Expr, name string) {
	suggestion, _ := searchquery.SuggestName(name, getMemberFunctions())
	w.problems = append(w.problems, searchquery.Problem{
		Position: getExprPosition(w.text, w.source, expr), Message: "Unknown function " + name, Suggestion: suggestion,
	})
}

func getMemberFunctions() searchquery.Catalog {
	return searchquery.Catalog{Functions: []searchquery.Function{
		{Name: overloads.StartsWith}, {Name: overloads.EndsWith}, {Name: overloads.Contains},
	}}
}

func (w *nameWalk) walk(expr ast.Expr) {
	switch expr.Kind() {
	case ast.CallKind:
		w.walkCall(expr)
	case ast.IdentKind:
		if f, ok := w.catalog.FindFunction(expr.AsIdent()); !ok || !f.IsVariable {
			w.addProblem(expr, "Unknown name", expr.AsIdent())
		}
	case ast.SelectKind:
		w.walk(expr.AsSelect().Operand())
	case ast.ListKind:
		for _, element := range expr.AsList().Elements() {
			w.walk(element)
		}
	case ast.UnspecifiedExprKind, ast.ComprehensionKind, ast.LiteralKind, ast.MapKind, ast.StructKind:
	}
}

func (w *nameWalk) walkCall(expr ast.Expr) {
	call := expr.AsCall()
	name := call.FunctionName()
	switch {
	case isOperator(name):
	case call.IsMemberFunction() && isStringMethod(name):
		w.walk(call.Target())
	case call.IsMemberFunction() && w.hasChunkCall(call.Target()):
		w.walk(call.Target())
		w.problems = append(w.problems, searchquery.Problem{
			Position: getExprPosition(w.text, w.source, expr), Message: "Unknown stage " + name,
		})
		return
	case call.IsMemberFunction():
		w.walk(call.Target())
		w.addMemberProblem(expr, name)
		return
	default:
		if f, ok := w.catalog.FindFunction(name); !ok || f.IsVariable {
			w.addProblem(expr, "Unknown function", name)
			return
		}
	}
	for _, arg := range call.Args() {
		w.walk(arg)
	}
}

func isOperator(name string) bool {
	_, found := operators.FindReverse(name)
	return found || name == operators.NotStrictlyFalse
}

func (w *nameWalk) hasChunkCall(expr ast.Expr) bool {
	if expr.Kind() != ast.CallKind {
		return false
	}
	call := expr.AsCall()
	if f, ok := w.catalog.FindFunction(call.FunctionName()); ok && f.Level == searchquery.LevelChunk && !call.IsMemberFunction() {
		return true
	}
	if call.IsMemberFunction() && w.hasChunkCall(call.Target()) {
		return true
	}
	return slices.ContainsFunc(call.Args(), w.hasChunkCall)
}

func isStringMethod(name string) bool {
	return name == overloads.StartsWith || name == overloads.EndsWith || name == overloads.Contains
}
