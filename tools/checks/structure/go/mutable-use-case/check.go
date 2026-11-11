package mutableusecase

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
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
		for _, target := range load.GetAssignedTargets(node) {
			if _, isSelector := target.(*ast.SelectorExpr); !isSelector {
				continue
			}
			if root := load.GetRootIdent(target); root != nil && info.Uses[root] == info.Defs[receiver] {
				isAssigned = true
			}
		}
		return !isAssigned
	})
	return isAssigned
}

func listMutatingWithMethods(pkgs []*packages.Package) (found []load.Finding, useCases int) {
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
					found = append(found, load.NewFinding(pkg.Fset, function.Pos(), named.Obj().Name()+"."+function.Name.Name+" changes the use case after construction"))
				}
			}
		}
	}
	return load.SortFindings(found), useCases
}

func isConstructorName(name string) bool {
	return strings.HasPrefix(name, "New") || strings.HasPrefix(name, "new")
}

func isFunctionField(field *types.Var) bool {
	_, isSignature := field.Type().Underlying().(*types.Signature)
	return field.IsField() && field.Exported() && isSignature
}

func listLateFunctionFields(pkgs []*packages.Package) (found []load.Finding, fields int) {
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
							found = append(found, load.NewFinding(pkg.Fset, target.Pos(), "assigns the function field "+field.Name()+" after construction"))
						}
					}
					return true
				})
			}
		}
	}
	return load.SortFindings(found), fields
}

func listWorkspaceProductionPackages(t *testing.T, workspace load.WorkspacePackages) []*packages.Package {
	t.Helper()
	var all []*packages.Package
	for _, dir := range load.ListWorkspaceModules(t) {
		all = append(all, load.ListProductionPackages(workspace.ListModule(t, dir))...)
	}
	return all
}

func checkNoUseCaseHasAMutatingWithMethod(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	found, useCases := listMutatingWithMethods(listWorkspaceProductionPackages(t, workspace))
	for _, one := range found {
		t.Errorf("%s: %s; a use case is complete when its constructor returns", one.Position, one.Name)
	}
	if useCases == 0 {
		t.Fatal("no use case read: the rule is checked against nothing")
	}
}

func checkNoFunctionFieldIsAssignedAfterConstruction(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	found, fields := listLateFunctionFields(listWorkspaceProductionPackages(t, workspace))
	for _, one := range found {
		t.Errorf("%s: %s; a struct is complete when its constructor returns", one.Position, one.Name)
	}
	if fields == 0 {
		t.Fatal("no exported function field of a use case or adapter read: the rule is checked against nothing")
	}
}
