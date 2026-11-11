package structusecases

import (
	"go/ast"
	"go/types"
	"slices"
	"strconv"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// maxUseCasesInStruct is the most use cases one struct holds, as fields of a use case type or as function
// fields that receive a method value of a use case.
const maxUseCasesInStruct = 5

func isUseCaseType(t types.Type) bool {
	if pointer, ok := types.Unalias(t).(*types.Pointer); ok {
		t = pointer.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	return strings.Contains(named.Obj().Pkg().Path()+"/", "/usecase/")
}

// structUseCaseCount is what one walk over structs read.
type structUseCaseCount struct {
	structs  int
	useCases int
}

func getCrowdedStructs(pkgs []*packages.Package, limit int) ([]load.Finding, structUseCaseCount) {
	var found []load.Finding
	var count structUseCaseCount
	holders := getUseCaseMethodHolders(pkgs)
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			if ast.IsGenerated(file) {
				continue
			}
			ast.Inspect(file, func(node ast.Node) bool {
				spec, ok := node.(*ast.TypeSpec)
				if !ok {
					return true
				}
				object, ok := pkg.TypesInfo.Defs[spec.Name].(*types.TypeName)
				if !ok || object.IsAlias() {
					return true
				}
				shape, ok := object.Type().Underlying().(*types.Struct)
				if !ok {
					return true
				}
				count.structs++
				inStruct := 0
				for field := range shape.Fields() {
					if isUseCaseType(field.Type()) || holders[field] {
						inStruct++
					}
				}
				count.useCases += inStruct
				if inStruct > limit {
					found = append(found, load.NewFinding(pkg.Fset, spec.Name.Pos(),
						pkg.Types.Name()+"."+object.Name()+" holds "+strconv.Itoa(inStruct)+" use cases"))
				}
				return true
			})
		}
	}
	return load.SortFindings(found), count
}

func checkNoStructHoldsMoreThanFiveUseCases(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	var total structUseCaseCount
	for _, dir := range load.ListWorkspaceModules(t) {
		found, count := getCrowdedStructs(load.ListProductionPackages(workspace.ListModule(t, dir)), maxUseCasesInStruct)
		total.structs += count.structs
		total.useCases += count.useCases
		for _, one := range found {
			t.Errorf("%s: %s, more than the %d a struct may hold", one.Position, one.Name, maxUseCasesInStruct)
		}
	}

	if total.structs == 0 || total.useCases == 0 {
		t.Fatalf("%d structs and %d use case fields read: the rule is checked against nothing", total.structs, total.useCases)
	}
}

func isUseCaseMethodValue(info *types.Info, expr ast.Expr) bool {
	selector, ok := ast.Unparen(expr).(*ast.SelectorExpr)
	if !ok {
		return false
	}
	selection, ok := info.Selections[selector]
	return ok && selection.Kind() == types.MethodVal && isUseCaseType(selection.Recv())
}

func getStructField(info *types.Info, literal *ast.CompositeLit, index int, element ast.Expr) *types.Var {
	shape, ok := info.TypeOf(literal).Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	if keyed, isKeyed := element.(*ast.KeyValueExpr); isKeyed {
		key, isIdent := keyed.Key.(*ast.Ident)
		if !isIdent {
			return nil
		}
		field, _ := info.Uses[key].(*types.Var)
		return field
	}
	if index < shape.NumFields() {
		return shape.Field(index)
	}
	return nil
}

func getUseCaseMethodHolders(pkgs []*packages.Package) map[*types.Var]bool {
	holders := map[*types.Var]bool{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				switch one := node.(type) {
				case *ast.CompositeLit:
					if pkg.TypesInfo.TypeOf(one) == nil {
						return true
					}
					for index, element := range one.Elts {
						value := element
						if keyed, ok := element.(*ast.KeyValueExpr); ok {
							value = keyed.Value
						}
						if field := getStructField(pkg.TypesInfo, one, index, element); field != nil && isUseCaseMethodValue(pkg.TypesInfo, value) {
							holders[field] = true
						}
					}
				case *ast.AssignStmt:
					for index, target := range one.Lhs {
						selector, ok := target.(*ast.SelectorExpr)
						if !ok || index >= len(one.Rhs) || !isUseCaseMethodValue(pkg.TypesInfo, one.Rhs[index]) {
							continue
						}
						if selection, ok := pkg.TypesInfo.Selections[selector]; ok && selection.Kind() == types.FieldVal {
							if field, ok := selection.Obj().(*types.Var); ok {
								holders[field] = true
							}
						}
					}
				}
				return true
			})
		}
	}
	return holders
}
