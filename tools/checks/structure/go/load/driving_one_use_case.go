package load

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// IsDrivingAdapter reports whether pkg is an adapter of the core that imports a use case package: whatever
// calls a use case drives the core, wherever its folder lies below adapter/ and internal/adapter/.
func IsDrivingAdapter(pkg *packages.Package, root string) bool {
	if !IsInLayer(pkg.PkgPath, root, "adapter") && !IsInLayer(pkg.PkgPath, root, "internal/adapter") {
		return false
	}
	for path := range pkg.Imports {
		if IsInLayer(path, root, "usecase") {
			return true
		}
	}
	return false
}

// ListExecuteSignatures returns the signatures of the Execute methods of the named types that the use case
// packages imported by pkg declare. A signature without parameters and results is left out: every callback
// has it.
func ListExecuteSignatures(pkg *packages.Package, root string) []*types.Signature {
	var found []*types.Signature
	for path, imported := range pkg.Imports {
		if IsInLayer(path, root, "usecase") && imported.Types != nil {
			found = append(found, listPackageExecuteSignatures(imported.Types)...)
		}
	}
	return found
}

func listPackageExecuteSignatures(pkg *types.Package) []*types.Signature {
	var found []*types.Signature
	for _, name := range pkg.Scope().Names() {
		object, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok || types.IsInterface(object.Type()) {
			continue
		}
		for method := range types.NewMethodSet(types.NewPointer(object.Type())).Methods() {
			signature, ok := method.Obj().Type().(*types.Signature)
			if ok && strings.HasPrefix(method.Obj().Name(), "Execute") && (signature.Params().Len() > 0 || signature.Results().Len() > 0) {
				found = append(found, signature)
			}
		}
	}
	return found
}

// IsExecuteFunctionField reports whether a field holds a function of the signature of a use case's Execute:
// the use case reached as a function value.
func IsExecuteFunctionField(field *types.Var, signatures []*types.Signature) bool {
	signature, ok := types.Unalias(field.Type()).(*types.Signature)
	return ok && slices.ContainsFunc(signatures, func(one *types.Signature) bool { return types.Identical(one, signature) })
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

// FunctionUse is what one function of a package reaches: the use case fields it selects a method of, the
// fields that hold a use case as a function, the Execute methods it takes as values, called
// or passed on, and the functions of the same package it calls.
type FunctionUse struct {
	Position Finding
	Direct   []types.Object
	Callees  []*types.Func
}

func (u *FunctionUse) Add(field *types.Var) {
	if field != nil {
		u.AddObject(field)
	}
}

// AddObject records a use case reached: a field, or the Execute method of a use case that no field holds.
func (u *FunctionUse) AddObject(object types.Object) {
	if !slices.Contains(u.Direct, object) {
		u.Direct = append(u.Direct, object)
	}
}

func GetFunctionUses(pkg *packages.Package, root string) map[*types.Func]*FunctionUse {
	uses := map[*types.Func]*FunctionUse{}
	signatures := ListExecuteSignatures(pkg, root)
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
			uses[object] = readFunctionUse(pkg, function, root, signatures)
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

func readFunctionUse(pkg *packages.Package, function *ast.FuncDecl, root string, signatures []*types.Signature) *FunctionUse {
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
			readFunctionValueUse(pkg.TypesInfo, one, root, signatures, use)
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

func GetReach(uses map[*types.Func]*FunctionUse, start *types.Func) []types.Object {
	var reach []types.Object
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
		if !IsDrivingAdapter(pkg, root) {
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

func getManyCallFinding(uses map[*types.Func]*FunctionUse, object *types.Func, reach []types.Object) (Finding, bool) {
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
		names[i] = getReachName(field)
	}
	slices.Sort(names)
	position := use.Position
	position.Name += " calls " + strings.Join(names, ", ")
	return position, true
}

func readFunctionValueUse(info *types.Info, expr *ast.SelectorExpr, root string, signatures []*types.Signature, use *FunctionUse) {
	selection, ok := info.Selections[expr]
	if !ok {
		return
	}
	switch object := selection.Obj().(type) {
	case *types.Var:
		if IsExecuteFunctionField(object, signatures) {
			use.Add(object)
		}
	case *types.Func:
		if !strings.HasPrefix(object.Name(), "Execute") || object.Pkg() == nil || !IsInLayer(object.Pkg().Path(), root, "usecase") {
			return
		}
		if GetUsedField(info, expr.X, root) == nil {
			use.AddObject(object)
		}
	}
}

func getReachName(object types.Object) string {
	function, ok := object.(*types.Func)
	if !ok {
		return object.Name()
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return function.Name()
	}
	receiver := types.Unalias(signature.Recv().Type())
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	if named, ok := receiver.(*types.Named); ok {
		return named.Obj().Pkg().Name() + "." + named.Obj().Name() + "." + function.Name()
	}
	return function.Name()
}
