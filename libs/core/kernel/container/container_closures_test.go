package container

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"testing"

	"golang.org/x/tools/go/packages"
)

func getRootIdent(expr ast.Expr) *ast.Ident {
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

func getAssignedTargets(node ast.Node) []ast.Expr {
	switch one := node.(type) {
	case *ast.AssignStmt:
		return one.Lhs
	case *ast.IncDecStmt:
		return []ast.Expr{one.X}
	}
	return nil
}

// listCapturedAssignments lists the assignments inside a function literal to a variable, or to a field or
// element of a variable, that the literal captures from the function around it.
func listCapturedAssignments(pkg *packages.Package) []finding {
	var found []finding
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
				for _, target := range getAssignedTargets(inner) {
					found = appendCaptured(found, pkg, literal, target)
				}
				return true
			})
			return true
		})
	}
	return sortFindings(found)
}

func appendCaptured(found []finding, pkg *packages.Package, literal *ast.FuncLit, target ast.Expr) []finding {
	root := getRootIdent(target)
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
	return append(found, newFinding(pkg.Fset, target.Pos(), "assigns "+root.Name+", a variable the function literal captures"))
}

// listGenericFunctions lists the functions with a type parameter. The container wires concrete types, so a
// type parameter there stands in for a type that the function avoids importing.
func listGenericFunctions(pkg *packages.Package) (found []finding, functions int) {
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
				found = append(found, newFinding(pkg.Fset, function.Pos(), function.Name.Name+" has a type parameter"))
			}
		}
	}
	return sortFindings(found), functions
}

func getContainerPackage(pkgs []*packages.Package) *packages.Package {
	for _, pkg := range pkgs {
		if pkg.PkgPath == coreImportPath+"/container" {
			return pkg
		}
	}
	return nil
}

func checkTheContainerAssignsNoCapturedVariable(t *testing.T, workspace workspacePackages) {
	t.Helper()
	pkg := getContainerPackage(getModulePackages(t, workspace, "libs/core/kernel"))
	if pkg == nil {
		t.Fatal("the container package is not loaded: the rule is checked against nothing")
	}
	for _, one := range listCapturedAssignments(pkg) {
		t.Errorf("%s: %s; the container wires values and a closure does not fill them in later", one.position, one.name)
	}
}

func checkTheContainerDeclaresNoGenericFunction(t *testing.T, workspace workspacePackages) {
	t.Helper()
	pkg := getContainerPackage(getModulePackages(t, workspace, "libs/core/kernel"))
	if pkg == nil {
		t.Fatal("the container package is not loaded: the rule is checked against nothing")
	}
	found, functions := listGenericFunctions(pkg)
	for _, one := range found {
		t.Errorf("%s: %s; the container names the types it wires", one.position, one.name)
	}
	if functions == 0 {
		t.Fatal("no function of the container read: the rule is checked against nothing")
	}
}

func TestCapturedAssignmentRule_RejectsAClosureAssigningACapturedVariable(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"container/wire.go": `package container

type Config struct{ Versions []string }

var global int

func Wire(c *Config) func() {
	count := 0
	local := func() {
		inner := 0
		inner++
		var other []int
		other = append(other, inner)
		_ = other
	}
	local()
	return func() {
		c.Versions = nil
		count++
		global = 1
		a, b := 1, 2
		a = b
		count, a = a, count
	}
}
`,
	})
	found := listCapturedAssignments(loadFixture(t, dir)[0])
	want := []string{
		"wire.go:18 assigns c, a variable the function literal captures",
		"wire.go:19 assigns count, a variable the function literal captures",
		"wire.go:23 assigns count, a variable the function literal captures",
	}
	if got := formatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

func TestGenericFunctionRule_RejectsAFunctionWithATypeParameter(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"container/wire.go": `package container

func Set[T any](target *T, value T) { *target = value }

func Plain(target *int) { *target = 1 }

type Box[T any] struct{ value T }

func (b Box[T]) Get() T { return b.value }
`,
	})
	found, functions := listGenericFunctions(loadFixture(t, dir)[0])
	want := []string{"wire.go:3 Set has a type parameter"}
	if got := formatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if functions != 3 {
		t.Errorf("%d functions read, want 3", functions)
	}
}
