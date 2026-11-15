package celquery

import (
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
	suggestion, _ := searchquery.SuggestName(name, memberFunctions)
	w.problems = append(w.problems, searchquery.Problem{
		Position: getExprPosition(w.text, w.source, expr), Message: "Unknown function " + name, Suggestion: suggestion,
	})
}

var memberFunctions = searchquery.Catalog{Functions: []searchquery.Function{
	{Name: overloads.StartsWith}, {Name: overloads.EndsWith}, {Name: overloads.Contains},
}}

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
	case call.IsMemberFunction():
		if !w.walkMember(expr) {
			return
		}
	default:
		if f, ok := w.catalog.FindFunction(name); !ok || f.IsVariable || f.IsStage {
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
	return found || name == operators.NotStrictlyFalse || name == operators.OldNotStrictlyFalse
}

func (w *nameWalk) isStage(name string) bool {
	f, ok := w.catalog.FindFunction(name)
	return ok && f.IsStage
}

func (w *nameWalk) walkMember(expr ast.Expr) bool {
	call := expr.AsCall()
	name := call.FunctionName()
	switch {
	case isStringMethod(name), w.isStage(name):
		w.walk(call.Target())
		return true
	case call.Target().Kind() == ast.CallKind:
		w.walk(call.Target())
		suggestion, _ := searchquery.SuggestStage(name, w.catalog)
		w.problems = append(w.problems, searchquery.Problem{
			Position: getExprPosition(w.text, w.source, expr), Message: "Unknown stage " + name, Suggestion: suggestion,
		})
		return false
	}
	w.walk(call.Target())
	w.addMemberProblem(expr, name)
	return false
}

func isStringMethod(name string) bool {
	return name == overloads.StartsWith || name == overloads.EndsWith || name == overloads.Contains
}
