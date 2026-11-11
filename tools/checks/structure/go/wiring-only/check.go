package wiringonly

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strconv"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// maxWiringComplexity is the cyclomatic complexity of the most complex function of the composition root: a
// function that selects and connects takes at most two decisions, such as a guard and a loop-free choice.
// Anything that decides more over data is a rule, and a rule belongs to a use case or an adapter.
const maxWiringComplexity = 3

// minWiringFunctions is how many functions the walk of the composition root reads at least; fewer means the
// check read the wrong package.
const minWiringFunctions = 100

func isRandomImport(path string) bool {
	return load.IsWithin(path, "math/rand") || load.IsWithin(path, "crypto")
}

func isPrintCall(fn *types.Func) bool {
	return fn.Pkg() != nil && fn.Pkg().Path() == "fmt" && (strings.HasPrefix(fn.Name(), "Fprint") || strings.HasPrefix(fn.Name(), "Print"))
}

func getComplexity(fn *ast.FuncDecl) int {
	complexity := 1
	ast.Inspect(fn, func(node ast.Node) bool {
		switch one := node.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			if one.List != nil {
				complexity++
			}
		case *ast.CommClause:
			if one.Comm != nil {
				complexity++
			}
		case *ast.BinaryExpr:
			if one.Op == token.LAND || one.Op == token.LOR {
				complexity++
			}
		}
		return true
	})
	return complexity
}

func getWiringFindings(pkgs []*packages.Package, path string) (found []load.Finding, functions int) {
	for _, pkg := range pkgs {
		if pkg.PkgPath != path {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, spec := range file.Imports {
				imported, err := strconv.Unquote(spec.Path.Value)
				if err == nil && (load.IsTechnologyImport(imported) || isRandomImport(imported)) {
					found = append(found, load.NewFinding(pkg.Fset, spec.Pos(), "imports "+imported))
				}
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, isCall := node.(*ast.CallExpr)
				if !isCall {
					return true
				}
				if fn, isFunc := typeutil.Callee(pkg.TypesInfo, call).(*types.Func); isFunc && isPrintCall(fn) {
					found = append(found, load.NewFinding(pkg.Fset, call.Pos(), "calls fmt."+fn.Name()))
				}
				return true
			})
			for _, decl := range file.Decls {
				fn, isFn := decl.(*ast.FuncDecl)
				if !isFn || fn.Body == nil {
					continue
				}
				functions++
				if complexity := getComplexity(fn); complexity > maxWiringComplexity {
					found = append(found, load.NewFinding(pkg.Fset, fn.Pos(), fn.Name.Name+" has complexity "+strconv.Itoa(complexity)))
				}
			}
		}
	}
	return load.SortFindings(found), functions
}

func checkTheCompositionRootHoldsWiringOnly(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	found, functions := getWiringFindings(load.GetModulePackages(t, workspace, "libs/core/kernel"), load.CoreImportPath+"/container")
	for _, one := range found {
		t.Errorf("%s: %s; the composition root connects and holds no rule", one.Position, one.Name)
	}
	if functions < minWiringFunctions {
		t.Fatalf("%d functions of the composition root read: the rule is checked against nothing", functions)
	}
}
