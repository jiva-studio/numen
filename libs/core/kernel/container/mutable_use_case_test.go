package container

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func isUseCaseOrAdapterPath(path string) bool {
	return strings.Contains(path+"/", "/usecase/") || strings.Contains(path+"/", "/adapter/")
}

func isUseCaseMethodSet(named *types.Named) bool {
	for method := range types.NewMethodSet(types.NewPointer(named)).Methods() {
		if strings.HasPrefix(method.Obj().Name(), "Execute") {
			return true
		}
	}
	return false
}

func getReceiverIdent(function *ast.FuncDecl) *ast.Ident {
	if function.Recv == nil || len(function.Recv.List) == 0 || len(function.Recv.List[0].Names) == 0 {
		return nil
	}
	return function.Recv.List[0].Names[0]
}

func isAssigningThroughReceiver(function *ast.FuncDecl, info *types.Info, receiver *ast.Ident) bool {
	if receiver == nil {
		return false
	}
	isAssigned := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		for _, target := range getAssignedTargets(node) {
			if _, isSelector := target.(*ast.SelectorExpr); !isSelector {
				continue
			}
			if root := getRootIdent(target); root != nil && info.Uses[root] == info.Defs[receiver] {
				isAssigned = true
			}
		}
		return !isAssigned
	})
	return isAssigned
}

// listMutatingWithMethods lists the methods named With... of use case types that assign to the receiver or
// that return the receiver's type, which is a modified copy.
func listMutatingWithMethods(pkgs []*packages.Package) (found []finding, useCases int) {
	for _, pkg := range pkgs {
		if !strings.Contains(pkg.PkgPath+"/", "/usecase/") {
			continue
		}
		for _, file := range pkg.Syntax {
			if ast.IsGenerated(file) {
				continue
			}
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv == nil || function.Body == nil {
					continue
				}
				object, ok := pkg.TypesInfo.Defs[function.Name].(*types.Func)
				if !ok {
					continue
				}
				signature, ok := object.Type().(*types.Signature)
				if !ok {
					continue
				}
				receiverType := signature.Recv().Type()
				if pointer, isPointer := receiverType.(*types.Pointer); isPointer {
					receiverType = pointer.Elem()
				}
				named, ok := receiverType.(*types.Named)
				if !ok || !isUseCaseMethodSet(named) {
					continue
				}
				if function.Name.Name == "Execute" {
					useCases++
				}
				if !strings.HasPrefix(function.Name.Name, "With") {
					continue
				}
				isCopy := false
				for result := range signature.Results().Variables() {
					resultType := result.Type()
					if pointer, isPointer := resultType.(*types.Pointer); isPointer {
						resultType = pointer.Elem()
					}
					isCopy = isCopy || types.Identical(resultType, named)
				}
				if isCopy || isAssigningThroughReceiver(function, pkg.TypesInfo, getReceiverIdent(function)) {
					found = append(found, newFinding(pkg.Fset, function.Pos(), named.Obj().Name()+"."+function.Name.Name+" changes the use case after construction"))
				}
			}
		}
	}
	return sortFindings(found), useCases
}

func isConstructorName(name string) bool {
	return strings.HasPrefix(name, "New") || strings.HasPrefix(name, "new")
}

func isFunctionField(field *types.Var) bool {
	_, isSignature := field.Type().Underlying().(*types.Signature)
	return field.IsField() && field.Exported() && isSignature
}

// listLateFunctionFields lists the assignments x.Field = value, outside a constructor, to an exported
// function-typed field of a struct declared in a use case or adapter package.
func listLateFunctionFields(pkgs []*packages.Package) (found []finding, fields int) {
	for _, pkg := range pkgs {
		if isUseCaseOrAdapterPath(pkg.PkgPath) {
			for _, name := range pkg.Types.Scope().Names() {
				object, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
				if !ok || object.IsAlias() {
					continue
				}
				if shape, isStruct := object.Type().Underlying().(*types.Struct); isStruct {
					for field := range shape.Fields() {
						if isFunctionField(field) {
							fields++
						}
					}
				}
			}
		}
		for _, file := range pkg.Syntax {
			if ast.IsGenerated(file) {
				continue
			}
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Body == nil || function.Recv == nil && isConstructorName(function.Name.Name) {
					continue
				}
				ast.Inspect(function.Body, func(node ast.Node) bool {
					assign, isAssign := node.(*ast.AssignStmt)
					if !isAssign || assign.Tok != token.ASSIGN {
						return true
					}
					for _, target := range assign.Lhs {
						selector, isSelector := target.(*ast.SelectorExpr)
						if !isSelector {
							continue
						}
						selection, isSelected := pkg.TypesInfo.Selections[selector]
						if !isSelected || selection.Kind() != types.FieldVal {
							continue
						}
						field, isVar := selection.Obj().(*types.Var)
						if isVar && isFunctionField(field) && field.Pkg() != nil && isUseCaseOrAdapterPath(field.Pkg().Path()) {
							found = append(found, newFinding(pkg.Fset, target.Pos(), "assigns the function field "+field.Name()+" after construction"))
						}
					}
					return true
				})
			}
		}
	}
	return sortFindings(found), fields
}

func listWorkspaceProductionPackages(t *testing.T, workspace workspacePackages) []*packages.Package {
	t.Helper()
	var all []*packages.Package
	for _, dir := range listWorkspaceModules(t) {
		all = append(all, listProductionPackages(workspace.listModule(t, dir))...)
	}
	return all
}

func checkNoUseCaseHasAMutatingWithMethod(t *testing.T, workspace workspacePackages) {
	t.Helper()
	found, useCases := listMutatingWithMethods(listWorkspaceProductionPackages(t, workspace))
	for _, one := range found {
		t.Errorf("%s: %s; a use case is complete when its constructor returns", one.position, one.name)
	}
	if useCases == 0 {
		t.Fatal("no use case read: the rule is checked against nothing")
	}
}

func checkNoFunctionFieldIsAssignedAfterConstruction(t *testing.T, workspace workspacePackages) {
	t.Helper()
	found, fields := listLateFunctionFields(listWorkspaceProductionPackages(t, workspace))
	for _, one := range found {
		t.Errorf("%s: %s; a struct is complete when its constructor returns", one.position, one.name)
	}
	if fields == 0 {
		t.Fatal("no exported function field of a use case or adapter read: the rule is checked against nothing")
	}
}

func TestWithMethodRule_RejectsAUseCaseChangedAfterConstruction(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"usecase/find/find.go": `package find

type Find struct{ limit int }

func (*Find) Execute() {}

func (f *Find) WithLimit(limit int) { f.limit = limit }

func (f Find) WithCopy(limit int) Find { f.limit = limit; return f }

func (f *Find) WithName(name string) string { return name }

func (f *Find) Limit() int { return f.limit }

type Helper struct{ limit int }

func (h *Helper) WithLimit(limit int) { h.limit = limit }
`,
	})
	found, useCases := listMutatingWithMethods(loadFixture(t, dir))
	want := []string{
		"find.go:7 Find.WithLimit changes the use case after construction",
		"find.go:9 Find.WithCopy changes the use case after construction",
	}
	if got := formatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if useCases != 1 {
		t.Errorf("%d use cases read, want 1", useCases)
	}
}

func TestFunctionFieldRule_RejectsAnAssignmentAfterConstruction(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"adapter/store/store.go": `package store

type Store struct {
	Notify func()
	hidden func()
	Name   string
}

func NewStore() *Store {
	store := &Store{}
	store.Notify = func() {}
	return store
}

func (s *Store) Attach(notify func()) { s.Notify = notify }

func (s *Store) Rename(name string) { s.Name = name; s.hidden = nil }
`,
		"container/wire.go": `package container

import "fixture/adapter/store"

func Wire() *store.Store {
	built := store.NewStore()
	built.Notify = nil
	return built
}
`,
	})
	found, fields := listLateFunctionFields(loadFixture(t, dir))
	want := []string{
		"store.go:15 assigns the function field Notify after construction",
		"wire.go:7 assigns the function field Notify after construction",
	}
	if got := formatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if fields != 1 {
		t.Errorf("%d function fields read, want 1", fields)
	}
}

func TestConstructorsReportCounts(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks every module of the workspace")
	}
	workspace := loadWorkspace(t)
	kernelDir, err := filepath.Abs(filepath.Join(repoRootDir, "libs/core/kernel"))
	if err != nil {
		t.Fatal(err)
	}
	withError, withoutError := 0, 0
	for _, pkg := range listProductionPackages(workspace.listModule(t, kernelDir)) {
		if !isUseCaseOrAdapterPath(pkg.PkgPath) {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || !strings.HasPrefix(function.Name.Name, "New") {
					continue
				}
				results := function.Type.Results
				if results != nil && len(results.List) > 1 && types.Identical(pkg.TypesInfo.TypeOf(results.List[len(results.List)-1].Type), types.Universe.Lookup("error").Type()) {
					withError++
				} else {
					withoutError++
				}
			}
		}
	}
	t.Logf("constructors of use cases and adapters that return an error: %d, that do not: %d", withError, withoutError)
}
