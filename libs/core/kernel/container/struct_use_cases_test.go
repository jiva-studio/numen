package container

import (
	"go/ast"
	"go/types"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
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

func getCrowdedStructs(pkgs []*packages.Package, limit int) ([]finding, structUseCaseCount) {
	var found []finding
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
					found = append(found, newFinding(pkg.Fset, spec.Name.Pos(),
						pkg.Types.Name()+"."+object.Name()+" holds "+strconv.Itoa(inStruct)+" use cases"))
				}
				return true
			})
		}
	}
	return sortFindings(found), count
}

func checkNoStructHoldsMoreThanFiveUseCases(t *testing.T, workspace workspacePackages) {
	t.Helper()
	var total structUseCaseCount
	for _, dir := range listWorkspaceModules(t) {
		found, count := getCrowdedStructs(listProductionPackages(workspace.listModule(t, dir)), maxUseCasesInStruct)
		total.structs += count.structs
		total.useCases += count.useCases
		for _, one := range found {
			t.Errorf("%s: %s, more than the %d a struct may hold", one.position, one.name, maxUseCasesInStruct)
		}
	}

	if total.structs == 0 || total.useCases == 0 {
		t.Fatalf("%d structs and %d use case fields read: the rule is checked against nothing", total.structs, total.useCases)
	}
}

// The rule counts value, pointer and embedded fields of use case types, and not a field of another type.
func TestStructUseCaseRule_RejectsAStructOfMoreThanTheLimit(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"usecase/find/find.go":   "package find\n\ntype Find struct{}\n",
		"usecase/store/store.go": "package store\n\ntype Store struct{}\n",
		"adapter/window/window.go": `package window

import (
	"fixture/usecase/find"
	"fixture/usecase/store"
)

type Crowded struct {
	first  *find.Find
	second *find.Find
	third  *store.Store
	fourth *store.Store
	fifth  find.Find
	store.Store
}

type Full struct {
	first  *find.Find
	second *find.Find
	third  *store.Store
	fourth *store.Store
	fifth  *store.Store
	name   string
	names  []string
}
`,
	})
	found, count := getCrowdedStructs(loadFixture(t, dir), 5)
	if want := []string{"window.go:8 window.Crowded holds 6 use cases"}; !slices.Equal(formatFindings(found), want) {
		t.Errorf("the rule rejects %v, want %v", formatFindings(found), want)
	}
	if count.useCases != 11 {
		t.Errorf("%d use case fields read, want 11", count.useCases)
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

func TestStructUseCaseRule_CountsMethodValuesAndFunctionFields(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"usecase/find/find.go": `package find

type Find struct{}

func NewFind() *Find { return &Find{} }

func (*Find) Execute() {}

func (*Find) Look() {}
`,
		"adapter/window/window.go": `package window

import "fixture/usecase/find"

type Wired struct {
	one   func()
	two   func()
	three func()
	four  func()
	five  func()
	six   func()
	plain func()
}

func New(registry *find.Find) *Wired {
	wired := &Wired{
		one:   registry.Look,
		two:   find.NewFind().Execute,
		three: registry.Execute,
		plain: func() {},
	}
	wired.four = registry.Look
	return wired
}

func newPositional(registry *find.Find) Wired {
	return Wired{nil, nil, nil, nil, registry.Execute, registry.Look, nil}
}
`,
	})
	found, count := getCrowdedStructs(loadFixture(t, dir), 5)
	if want := []string{"window.go:5 window.Wired holds 6 use cases"}; !slices.Equal(formatFindings(found), want) {
		t.Errorf("the rule rejects %v, want %v", formatFindings(found), want)
	}
	if count.useCases != 6 {
		t.Errorf("%d use case fields read, want 6", count.useCases)
	}
}
