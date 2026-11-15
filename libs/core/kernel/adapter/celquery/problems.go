package celquery

import (
	"slices"
	"strings"

	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/overloads"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

// nameWalk finds every function, stage and variable of a parsed query that the catalog does not declare.
type nameWalk struct {
	catalog  searchquery.Catalog
	source   *ast.SourceInfo
	problems []searchquery.Problem
}

func (w *nameWalk) addProblem(expr ast.Expr, message, name string) {
	suggestion, _ := searchquery.SuggestName(name, w.catalog)
	w.problems = append(w.problems, searchquery.Problem{
		Position: getExprPosition(w.source, expr), Message: message + " " + name, Suggestion: suggestion,
	})
}

func (w *nameWalk) walk(expr ast.Expr) {
	switch expr.Kind() {
	case ast.CallKind:
		w.walkCall(expr)
	case ast.IdentKind:
		if f, ok := w.catalog.FindFunction(expr.AsIdent()); !ok || !f.IsVariable {
			w.addProblem(expr, "unknown name", expr.AsIdent())
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
	case call.IsMemberFunction() && name == overloads.StartsWith:
		w.walk(call.Target())
	case call.IsMemberFunction() && w.isStage(name):
		w.walk(call.Target())
	case call.IsMemberFunction() && (w.hasChunkCall(call.Target()) || isStageCall(call.Target()) || call.Target().Kind() == ast.CallKind):
		suggestion, _ := searchquery.SuggestStage(name, w.catalog)
		w.problems = append(w.problems, searchquery.Problem{
			Position: getExprPosition(w.source, expr), Message: "unknown stage " + name, Suggestion: suggestion,
		})
		return
	case call.IsMemberFunction():
		w.addProblem(expr, "unknown function", name)
		return
	default:
		if f, ok := w.catalog.FindFunction(name); !ok || f.IsVariable || f.IsStage {
			w.addProblem(expr, "unknown function", name)
			return
		}
	}
	for _, arg := range call.Args() {
		w.walk(arg)
	}
}

func isOperator(name string) bool {
	return strings.HasPrefix(name, "_") || strings.HasPrefix(name, "@") || strings.HasPrefix(name, "!")
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

func (w *nameWalk) isStage(name string) bool {
	f, ok := w.catalog.FindFunction(name)
	return ok && f.IsStage
}
