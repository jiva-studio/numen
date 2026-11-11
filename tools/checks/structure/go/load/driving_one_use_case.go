package load

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

func IsDrivingHandlerPackage(path, root string) bool {
	return IsInLayer(path, root, "adapter/window") || IsInLayer(path, root, "adapter/cli") ||
		IsInLayer(path, root, "adapter/plugin/hostendpoint") ||
		IsInLayer(path, root, "adapter/plugin/hostsearch") ||
		IsInLayer(path, root, "adapter/plugin/hostvault") ||
		IsInLayer(path, root, "adapter/plugin/hosttext") ||
		IsInLayer(path, root, "adapter/plugin/hoststore") ||
		IsInLayer(path, root, "adapter/plugin/hostqueue") ||
		IsInLayer(path, root, "adapter/plugin/hostwindow")
}

func IsUseCaseFieldType(t types.Type, root string) bool {
	if pointer, is := types.Unalias(t).(*types.Pointer); is {
		t = pointer.Elem()
	}
	named, is := types.Unalias(t).(*types.Named)
	if !is {
		return false
	}
	if iface, is := named.Underlying().(*types.Interface); is {
		for method := range iface.Methods() {
			if strings.HasPrefix(method.Name(), "Execute") {
				return true
			}
		}
		return false
	}
	if pkg := named.Obj().Pkg(); pkg == nil || !IsInLayer(pkg.Path(), root, "usecase") {
		return false
	}
	methods := types.NewMethodSet(types.NewPointer(named))
	for method := range methods.Methods() {
		if strings.HasPrefix(method.Obj().Name(), "Execute") {
			return true
		}
	}
	return false
}

func GetUseCaseField(info *types.Info, expr *ast.SelectorExpr, root string) *types.Var {
	selection, ok := info.Selections[expr]
	if !ok || selection.Kind() != types.FieldVal {
		return nil
	}
	field, is := selection.Obj().(*types.Var)
	if !is || !IsUseCaseFieldType(field.Type(), root) {
		return nil
	}
	return field
}

func GetUsedField(info *types.Info, expr ast.Expr, root string) *types.Var {
	if selector, is := expr.(*ast.SelectorExpr); is {
		return GetUseCaseField(info, selector, root)
	}
	return nil
}

// FunctionUse is what one function of a package reaches: the use case fields it selects a method of, called
// or passed on, and the functions of the same package it calls.
type FunctionUse struct {
	Position Finding
	Direct   []*types.Var
	Callees  []*types.Func
}

func (u *FunctionUse) Add(field *types.Var) {
	if field != nil && !slices.Contains(u.Direct, field) {
		u.Direct = append(u.Direct, field)
	}
}

func GetFunctionUses(pkg *packages.Package, root string) map[*types.Func]*FunctionUse {
	uses := map[*types.Func]*FunctionUse{}
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			function, is := declaration.(*ast.FuncDecl)
			if !is || function.Body == nil {
				continue
			}
			object, is := pkg.TypesInfo.Defs[function.Name].(*types.Func)
			if !is {
				continue
			}
			uses[object] = readFunctionUse(pkg, function, root)
		}
	}
	return uses
}

func listFunctionLiterals(body *ast.BlockStmt) []*ast.FuncLit {
	var literals []*ast.FuncLit
	ast.Inspect(body, func(node ast.Node) bool {
		if literal, is := node.(*ast.FuncLit); is {
			literals = append(literals, literal)
		}
		return true
	})
	return literals
}

func readFunctionUse(pkg *packages.Package, function *ast.FuncDecl, root string) *FunctionUse {
	use := &FunctionUse{Position: NewFinding(pkg.Fset, function.Pos(), function.Name.Name)}
	literals := listFunctionLiterals(function.Body)
	// A function literal runs when its caller decides, which is not when the function that builds it runs.
	isBuilt := func(position token.Pos) bool {
		return slices.ContainsFunc(literals, func(literal *ast.FuncLit) bool { return literal.Pos() <= position && position < literal.End() })
	}
	addCallee := func(name *ast.Ident, position token.Pos) {
		if called, is := pkg.TypesInfo.Uses[name].(*types.Func); is && called.Pkg() == pkg.Types && !isBuilt(position) {
			use.Callees = append(use.Callees, called)
		}
	}
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch one := node.(type) {
		case *ast.SelectorExpr:
			use.Add(GetUsedField(pkg.TypesInfo, one.X, root))
			addCallee(one.Sel, one.Pos())
		case *ast.CallExpr:
			for _, argument := range one.Args {
				use.Add(GetUsedField(pkg.TypesInfo, argument, root))
			}
		case *ast.Ident:
			addCallee(one, one.Pos())
		}
		return true
	})
	return use
}

func GetReach(uses map[*types.Func]*FunctionUse, start *types.Func) []*types.Var {
	var reach []*types.Var
	visited := map[*types.Func]bool{}
	var walk func(one *types.Func)
	walk = func(one *types.Func) {
		use, isDeclared := uses[one]
		if visited[one] || !isDeclared {
			return
		}
		visited[one] = true
		for _, field := range use.Direct {
			if !slices.Contains(reach, field) {
				reach = append(reach, field)
			}
		}
		for _, called := range use.Callees {
			walk(called)
		}
	}
	walk(start)
	return reach
}

func ListManyUseCaseCalls(pkgs []*packages.Package, root string) (found []Finding, functions int) {
	for _, pkg := range pkgs {
		if !IsDrivingHandlerPackage(pkg.PkgPath, root) {
			continue
		}
		uses := GetFunctionUses(pkg, root)
		for object := range uses {
			reach := GetReach(uses, object)
			if len(reach) == 0 {
				continue
			}
			functions++
			if finding, isMany := getManyCallFinding(uses, object, reach); isMany {
				found = append(found, finding)
			}
		}
	}
	return SortFindings(found), functions
}

func getManyCallFinding(uses map[*types.Func]*FunctionUse, object *types.Func, reach []*types.Var) (Finding, bool) {
	if len(reach) < 2 {
		return Finding{}, false
	}
	use := uses[object]
	// A function is named when no function it calls already reaches two: the one that joins them.
	delegated := slices.ContainsFunc(use.Callees, func(called *types.Func) bool {
		return called != object && len(GetReach(uses, called)) > 1
	})
	if delegated {
		return Finding{}, false
	}
	names := make([]string, len(reach))
	for i, field := range reach {
		names[i] = field.Name()
	}
	slices.Sort(names)
	position := use.Position
	position.Name += " calls " + strings.Join(names, ", ")
	return position, true
}
