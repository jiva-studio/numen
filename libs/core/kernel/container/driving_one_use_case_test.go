package container

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func isDrivingHandlerPackage(path, root string) bool {
	return isInLayer(path, root, "adapter/window") || isInLayer(path, root, "adapter/plugin/hostendpoint") ||
		isInLayer(path, root, "adapter/cli")
}

func isUseCaseFieldType(t types.Type, root string) bool {
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
	if pkg := named.Obj().Pkg(); pkg == nil || !isInLayer(pkg.Path(), root, "usecase") {
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

func getUseCaseField(info *types.Info, expr *ast.SelectorExpr, root string) *types.Var {
	selection, ok := info.Selections[expr]
	if !ok || selection.Kind() != types.FieldVal {
		return nil
	}
	field, is := selection.Obj().(*types.Var)
	if !is || !isUseCaseFieldType(field.Type(), root) {
		return nil
	}
	return field
}

func getUsedField(info *types.Info, expr ast.Expr, root string) *types.Var {
	if selector, is := expr.(*ast.SelectorExpr); is {
		return getUseCaseField(info, selector, root)
	}
	return nil
}

// functionUse is what one function of a package reaches: the use case fields it selects a method of, called
// or passed on, and the functions of the same package it calls.
type functionUse struct {
	position finding
	direct   []*types.Var
	callees  []*types.Func
}

func (u *functionUse) add(field *types.Var) {
	if field != nil && !slices.Contains(u.direct, field) {
		u.direct = append(u.direct, field)
	}
}

func getFunctionUses(pkg *packages.Package, root string) map[*types.Func]*functionUse {
	uses := map[*types.Func]*functionUse{}
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
			use := &functionUse{position: newFinding(pkg.Fset, function.Pos(), function.Name.Name)}
			var literals []*ast.FuncLit
			ast.Inspect(function.Body, func(node ast.Node) bool {
				if literal, is := node.(*ast.FuncLit); is {
					literals = append(literals, literal)
				}
				return true
			})
			// A function literal runs when its caller decides, which is not when the function that builds it runs.
			isBuilt := func(position token.Pos) bool {
				return slices.ContainsFunc(literals, func(literal *ast.FuncLit) bool { return literal.Pos() <= position && position < literal.End() })
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				switch one := node.(type) {
				case *ast.SelectorExpr:
					use.add(getUsedField(pkg.TypesInfo, one.X, root))
					if called, is := pkg.TypesInfo.Uses[one.Sel].(*types.Func); is && called.Pkg() == pkg.Types && !isBuilt(one.Pos()) {
						use.callees = append(use.callees, called)
					}
				case *ast.CallExpr:
					for _, argument := range one.Args {
						use.add(getUsedField(pkg.TypesInfo, argument, root))
					}
				case *ast.Ident:
					if called, is := pkg.TypesInfo.Uses[one].(*types.Func); is && called.Pkg() == pkg.Types && !isBuilt(one.Pos()) {
						use.callees = append(use.callees, called)
					}
				}
				return true
			})
			uses[object] = use
		}
	}
	return uses
}

func getReach(uses map[*types.Func]*functionUse, start *types.Func) []*types.Var {
	var reach []*types.Var
	visited := map[*types.Func]bool{}
	var walk func(one *types.Func)
	walk = func(one *types.Func) {
		use, isDeclared := uses[one]
		if visited[one] || !isDeclared {
			return
		}
		visited[one] = true
		for _, field := range use.direct {
			if !slices.Contains(reach, field) {
				reach = append(reach, field)
			}
		}
		for _, called := range use.callees {
			walk(called)
		}
	}
	walk(start)
	return reach
}

func listManyUseCaseCalls(pkgs []*packages.Package, root string) (found []finding, functions int) {
	for _, pkg := range pkgs {
		if !isDrivingHandlerPackage(pkg.PkgPath, root) {
			continue
		}
		uses := getFunctionUses(pkg, root)
		for object, use := range uses {
			reach := getReach(uses, object)
			if len(reach) == 0 {
				continue
			}
			functions++
			if len(reach) < 2 {
				continue
			}
			// A function is named when no function it calls already reaches two: the one that joins them.
			delegated := slices.ContainsFunc(use.callees, func(called *types.Func) bool {
				return called != object && len(getReach(uses, called)) > 1
			})
			if delegated {
				continue
			}
			names := make([]string, len(reach))
			for i, field := range reach {
				names[i] = field.Name()
			}
			slices.Sort(names)
			position := use.position
			position.name += " calls " + strings.Join(names, ", ")
			found = append(found, position)
		}
	}
	return sortFindings(found), functions
}

func checkDrivingFunctionCallsOneUseCase(t *testing.T, adapters []*packages.Package) {
	t.Helper()
	found, functions := listManyUseCaseCalls(adapters, coreImportPath)
	for _, one := range found {
		t.Errorf("%s %s: a driving adapter calls one use case", one.position, one.name)
	}
	// A walk that read few calls is a rule checked against almost nothing.
	if functions < 80 {
		t.Fatalf("%d driving-adapter functions reach a use case, want at least 80: the walk is not reading them", functions)
	}
}

func TestDrivingCallRule_RejectsAFunctionCallingTwoUseCases(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"usecase/find/find.go": "package find\n\ntype Find struct{}\n\nfunc (Find) Execute() {}\n\ntype Save struct{}\n\nfunc (Save) Execute() {}\n",
		"adapter/window/one/one.go": `package one

import "fixture/usecase/find"

type Reader interface{ Execute() }

type Helper struct{ save *find.Save }

func (h *Helper) Write() { h.save.Execute() }

type Handler struct {
	find   find.Find
	save   *find.Save
	reader Reader
	helper *Helper
	other  int
}

func (h *Handler) Both()    { h.find.Execute(); h.save.Execute() }
func (h *Handler) Passed()  { run(h.find.Execute, h.reader.Execute) }
func (h *Handler) Through() { h.find.Execute(); h.helper.Write() }
func (h *Handler) Above()   { h.Through() }
func (h *Handler) Again()   { h.find.Execute(); h.find.Execute() }
func (h *Handler) One()     { h.save.Execute() }
func (h *Handler) None()    { h.other++ }

func (h *Handler) Handed() { hand(h.find, h.save) }

func run(...func()) {}

func hand(...any) {}
`,
	})
	found, functions := listManyUseCaseCalls(loadFixture(t, dir), "fixture")
	want := []string{"one.go:19 Both calls find, save", "one.go:20 Passed calls find, reader", "one.go:21 Through calls find, save", "one.go:27 Handed calls find, save"}
	if !slices.Equal(formatFindings(found), want) {
		t.Errorf("the rule refuses %v, want %v", formatFindings(found), want)
	}
	if functions != 8 {
		t.Errorf("%d functions reach a use case, want 8", functions)
	}
}
