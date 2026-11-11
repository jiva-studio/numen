package load

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"testing"
	"golang.org/x/tools/go/packages"
)

func GetRootIdent(expr ast.Expr) *ast.Ident {
	for {
		switch one := expr.(type) {
		case *ast.Ident:
			return one
		case *ast.SelectorExpr:
			expr = one.X
		case *ast.IndexExpr:
			expr = one.X
		case *ast.StarExpr:
			expr = one.X
		case *ast.ParenExpr:
			expr = one.X
		default:
			return nil
		}
	}
}

func GetAssignedTargets(node ast.Node) []ast.Expr {
	switch one := node.(type) {
	case *ast.AssignStmt:
		return one.Lhs
	case *ast.IncDecStmt:
		return []ast.Expr{one.X}
	}
	return nil
}
