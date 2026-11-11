package containerclosures

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func listCapturedAssignments(pkg *packages.Package) []load.Finding {
	var found []load.Finding
	for _, file := range pkg.Syntax {
		if ast.IsGenerated(file) {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.FuncLit)
			if !ok {
				return true
			}
			ast.Inspect(literal.Body, func(inner ast.Node) bool {
				if assign, isAssign := inner.(*ast.AssignStmt); isAssign && assign.Tok == token.DEFINE {
					for _, target := range assign.Lhs {
						if ident, isIdent := target.(*ast.Ident); isIdent && pkg.TypesInfo.Defs[ident] != nil {
							continue
						}
						found = appendCaptured(found, pkg, literal, target)
					}
					return true
				}
				for _, target := range load.GetAssignedTargets(inner) {
					found = appendCaptured(found, pkg, literal, target)
				}
				return true
			})
			return true
		})
	}
	return load.SortFindings(found)
}

func appendCaptured(found []load.Finding, pkg *packages.Package, literal *ast.FuncLit, target ast.Expr) []load.Finding {
	root := load.GetRootIdent(target)
	if root == nil {
		return found
	}
	object, ok := pkg.TypesInfo.Uses[root].(*types.Var)
	if !ok || object.IsField() || object.Parent() == pkg.Types.Scope() {
		return found
	}
	if literal.Pos() <= object.Pos() && object.Pos() < literal.End() {
		return found
	}
	return append(found, load.NewFinding(pkg.Fset, target.Pos(), "assigns "+root.Name+", a variable the function literal captures"))
}

func listGenericFunctions(pkg *packages.Package) (found []load.Finding, functions int) {
	for _, file := range pkg.Syntax {
		if ast.IsGenerated(file) {
			continue
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			functions++
			if function.Type.TypeParams != nil && len(function.Type.TypeParams.List) > 0 {
				found = append(found, load.NewFinding(pkg.Fset, function.Pos(), function.Name.Name+" has a type parameter"))
			}
		}
	}
	return load.SortFindings(found), functions
}

func getContainerPackage(pkgs []*packages.Package) *packages.Package {
	for _, pkg := range pkgs {
		if pkg.PkgPath == load.CoreImportPath+"/container" {
			return pkg
		}
	}
	return nil
}

func checkTheContainerAssignsNoCapturedVariable(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	pkg := getContainerPackage(load.GetModulePackages(t, workspace, "libs/core/kernel"))
	if pkg == nil {
		t.Fatal("the container package is not loaded: the rule is checked against nothing")
	}
	for _, one := range listCapturedAssignments(pkg) {
		t.Errorf("%s: %s; the container wires values and a closure does not fill them in later", one.Position, one.Name)
	}
}

func checkTheContainerDeclaresNoGenericFunction(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	pkg := getContainerPackage(load.GetModulePackages(t, workspace, "libs/core/kernel"))
	if pkg == nil {
		t.Fatal("the container package is not loaded: the rule is checked against nothing")
	}
	found, functions := listGenericFunctions(pkg)
	for _, one := range found {
		t.Errorf("%s: %s; the container names the types it wires", one.Position, one.Name)
	}
	if functions == 0 {
		t.Fatal("no function of the container read: the rule is checked against nothing")
	}
}
