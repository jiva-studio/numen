package usecaseports

import (
	"go/ast"
	"go/types"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

func isUseCasePath(path string) bool {
	return strings.Contains(path+"/", "/usecase/") && !load.IsUseCaseTestSupport(path)
}

func isPortPath(path string) bool {
	return strings.Contains(path+"/", "/port/")
}

func isUseCaseMethod(name string) bool {
	return strings.HasPrefix(name, "Execute")
}

func isUseCaseType(named *types.Named) bool {
	for method := range types.NewMethodSet(types.NewPointer(named)).Methods() {
		if isUseCaseMethod(method.Obj().Name()) {
			return true
		}
	}
	return false
}

func listNamedTypes(pkg *packages.Package) []*types.Named {
	var found []*types.Named
	for _, name := range pkg.Types.Scope().Names() {
		object, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
		if !ok || object.IsAlias() {
			continue
		}
		if named, isNamed := object.Type().(*types.Named); isNamed && named.TypeParams().Len() == 0 {
			found = append(found, named)
		}
	}
	return found
}

func listUseCasesBehindInterfaces(pkgs []*packages.Package) (found []load.Finding, interfaces, useCases int) {
	var useCaseTypes []*types.Named
	type declared struct {
		named *types.Named
		shape *types.Interface
		pkg   *packages.Package
	}
	var ports []declared
	for _, pkg := range pkgs {
		for _, named := range listNamedTypes(pkg) {
			if shape, isInterface := named.Underlying().(*types.Interface); isInterface {
				if (isUseCasePath(pkg.PkgPath) || isPortPath(pkg.PkgPath)) && shape.NumMethods() > 0 {
					ports = append(ports, declared{named, shape, pkg})
				}
				continue
			}
			if isUseCasePath(pkg.PkgPath) && isUseCaseType(named) {
				useCaseTypes = append(useCaseTypes, named)
			}
		}
	}
	for _, port := range ports {
		for _, useCase := range useCaseTypes {
			if useCase.Obj().Pkg() == port.pkg.Types || !types.Implements(types.NewPointer(useCase), port.shape) {
				continue
			}
			found = append(found, load.NewFinding(port.pkg.Fset, port.named.Obj().Pos(), port.named.Obj().Name()+" is implemented by the use case "+useCase.Obj().Pkg().Name()+"."+useCase.Obj().Name()))
			break
		}
	}
	return load.SortFindings(found), len(ports), len(useCaseTypes)
}

func getUseCaseMethodValue(info *types.Info, expr ast.Expr) *types.Func {
	selector, ok := ast.Unparen(expr).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	selection, isSelected := info.Selections[selector]
	if !isSelected || selection.Kind() != types.MethodVal {
		return nil
	}
	method, isFunc := selection.Obj().(*types.Func)
	if !isFunc || method.Pkg() == nil || !isUseCasePath(method.Pkg().Path()) || !isUseCaseMethod(method.Name()) {
		return nil
	}
	return method
}

func listFunctionSeams(pkgs []*packages.Package) (found []load.Finding, values int) {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			if ast.IsGenerated(file) {
				continue
			}
			seen := map[*types.Var]*types.Func{}
			getValue := func(expr ast.Expr) *types.Func {
				if method := getUseCaseMethodValue(pkg.TypesInfo, expr); method != nil {
					return method
				}
				if ident, isIdent := ast.Unparen(expr).(*ast.Ident); isIdent {
					if variable, isVar := pkg.TypesInfo.Uses[ident].(*types.Var); isVar {
						return seen[variable]
					}
				}
				return nil
			}
			report := func(expr ast.Expr, destination *types.Package, what string) {
				method := getValue(expr)
				if method == nil || destination == nil || !isUseCasePath(destination.Path()) || destination == method.Pkg() {
					return
				}
				found = append(found, load.NewFinding(pkg.Fset, expr.Pos(), method.Pkg().Name()+"."+method.Name()+" is passed as a function to "+what+" of the use case package "+destination.Name()))
			}
			ast.Inspect(file, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.AssignStmt:
					for i, target := range node.Lhs {
						if len(node.Lhs) != len(node.Rhs) {
							break
						}
						if method := getUseCaseMethodValue(pkg.TypesInfo, node.Rhs[i]); method != nil {
							if ident, isIdent := target.(*ast.Ident); isIdent {
								if variable, isVar := pkg.TypesInfo.ObjectOf(ident).(*types.Var); isVar {
									seen[variable] = method
								}
							}
						}
						if selector, isSelector := target.(*ast.SelectorExpr); isSelector {
							if selection, isSelected := pkg.TypesInfo.Selections[selector]; isSelected && selection.Kind() == types.FieldVal {
								report(node.Rhs[i], selection.Obj().Pkg(), "the field "+selection.Obj().Name())
							}
						}
					}
				case *ast.CompositeLit:
					shape, isStruct := types.Unalias(pkg.TypesInfo.TypeOf(node)).Underlying().(*types.Struct)
					if !isStruct {
						return true
					}
					for i, element := range node.Elts {
						if pair, isPair := element.(*ast.KeyValueExpr); isPair {
							if key, isIdent := pair.Key.(*ast.Ident); isIdent {
								if field, isVar := pkg.TypesInfo.Uses[key].(*types.Var); isVar {
									report(pair.Value, field.Pkg(), "the field "+field.Name())
								}
							}
						} else if i < shape.NumFields() {
							report(element, shape.Field(i).Pkg(), "the field "+shape.Field(i).Name())
						}
					}
				case *ast.CallExpr:
					var callee types.Object
					switch function := ast.Unparen(node.Fun).(type) {
					case *ast.Ident:
						callee = pkg.TypesInfo.Uses[function]
					case *ast.SelectorExpr:
						callee = pkg.TypesInfo.Uses[function.Sel]
					}
					if function, isFunc := callee.(*types.Func); isFunc {
						for _, arg := range node.Args {
							report(arg, function.Pkg(), "the function "+function.Name())
						}
					}
				}
				return true
			})
			ast.Inspect(file, func(node ast.Node) bool {
				if expr, isExpr := node.(ast.Expr); isExpr && getUseCaseMethodValue(pkg.TypesInfo, expr) != nil {
					values++
				}
				return true
			})
		}
	}
	return load.SortFindings(found), values
}

func listKernelPackages(t *testing.T, workspace load.WorkspacePackages) []*packages.Package {
	t.Helper()
	return load.GetModulePackages(t, workspace, "libs/core/kernel")
}

func getKnownInterfaceExceptions() map[string]string {
	return map[string]string{
		"TriggerRunner is implemented by the use case trigger.Triggers": "waits for the window adapters of PR 8",
	}
}

func checkNoUseCaseIsBehindAnInterface(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	found, interfaces, useCases := listUseCasesBehindInterfaces(listKernelPackages(t, workspace))
	for _, one := range filterKnownExceptions(t, found) {
		t.Errorf("%s: %s; what use cases share is a rule of the domain or a port an adapter implements", one.Position, one.Name)
	}
	if interfaces == 0 || useCases == 0 {
		t.Fatal("no interface or no use case read: the rule is checked against nothing")
	}
}

func filterKnownExceptions(t *testing.T, found []load.Finding) []load.Finding {
	t.Helper()
	exceptions := getKnownInterfaceExceptions()
	seen := map[string]bool{}
	var rest []load.Finding
	for _, one := range found {
		if _, isKnown := exceptions[one.Name]; isKnown {
			seen[one.Name] = true
			continue
		}
		rest = append(rest, one)
	}
	for name := range exceptions {
		if !seen[name] {
			t.Errorf("the exception %q is no longer a finding: remove it from getKnownInterfaceExceptions", name)
		}
	}
	return rest
}

func checkNoUseCaseIsPassedAsAFunction(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	found, values := listFunctionSeams(listKernelPackages(t, workspace))
	for _, one := range found {
		t.Errorf("%s: %s; what use cases share is a rule of the domain or a port an adapter implements", one.Position, one.Name)
	}
	if values == 0 {
		t.Fatal("no method value of a use case read: the rule is checked against nothing")
	}
}
