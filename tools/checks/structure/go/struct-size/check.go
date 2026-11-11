package structsize

import (
	"go/ast"
	"go/types"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// structLimit is how many methods and fields one named type of a layer may have.
type structLimit struct {
	methods int
	fields  int
}

// The limits come from the target hexagon. A driven adapter wraps one technology behind two or three ports and
// a driving adapter serves one wire service of about ten calls, so twenty methods hold either with room;
// fifteen fields is what an adapter needs for the collaborators of one service (each a port of one to three
// methods) plus its state. A use case is one scenario: an Execute method, a few steps and the ports it
// requires, so twelve and twelve; a use case over that is cut by design. The composition root defines no object but Config. Embedded interfaces count the
// methods they give the type, and a function-typed field counts as a field, so a struct cannot meet the limit
// by embedding interfaces of five methods or by holding callbacks.
const (
	maxAdapterMethods = 20
	maxAdapterFields  = 15
	maxUseCaseMethods = 12
	maxUseCaseFields  = 12
)

func getStructLimit(rel string) (structLimit, bool) {
	switch {
	case load.IsWithin(rel, "usecase"):
		return structLimit{methods: maxUseCaseMethods, fields: maxUseCaseFields}, true
	case load.IsWithin(rel, "adapter"), load.IsWithin(rel, "internal/adapter"), load.IsWithin(rel, "container"):
		return structLimit{methods: maxAdapterMethods, fields: maxAdapterFields}, true
	}
	return structLimit{}, false
}

// maxConfigMethods is how many exported methods container.Config may have. Config is the one object the
// applications hold: it builds the wired parts and hands them out, and each exported method is a part
// the application reaches, so the number stays near what one application names.
const maxConfigMethods = 20

func getOversizedTypes(pkgs []*packages.Package, getLimit func(path string) (structLimit, bool)) ([]load.Finding, int) {
	var found []load.Finding
	read := 0
	for _, pkg := range pkgs {
		limit, ok := getLimit(pkg.PkgPath)
		if !ok {
			continue
		}
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
				if !ok || object.IsAlias() || types.IsInterface(object.Type()) {
					return true
				}
				named, ok := object.Type().(*types.Named)
				if !ok {
					return true
				}
				read++
				methods := types.NewMethodSet(types.NewPointer(named)).Len()
				fields := 0
				if st, ok := named.Underlying().(*types.Struct); ok {
					fields = countStructFields(st, map[*types.Struct]bool{})
				}
				if methods > limit.methods {
					found = append(found, load.NewFinding(pkg.Fset, spec.Name.Pos(),
						pkg.Types.Name()+"."+object.Name()+" has "+strconv.Itoa(methods)+" methods, more than "+strconv.Itoa(limit.methods)))
				}
				if fields > limit.fields {
					found = append(found, load.NewFinding(pkg.Fset, spec.Name.Pos(),
						pkg.Types.Name()+"."+object.Name()+" has "+strconv.Itoa(fields)+" fields, more than "+strconv.Itoa(limit.fields)))
				}
				return true
			})
		}
	}
	return load.SortFindings(found), read
}

func getExportedMethodCount(pkgs []*packages.Package, pkgPath, name string) (int, bool) {
	for _, pkg := range pkgs {
		if pkg.PkgPath != pkgPath {
			continue
		}
		object, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			return 0, false
		}
		set := types.NewMethodSet(types.NewPointer(object.Type()))
		count := 0
		for method := range set.Methods() {
			if method.Obj().Exported() {
				count++
			}
		}
		return count, true
	}
	return 0, false
}

func checkNoTypeExceedsItsLayersSize(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	pkgs := load.GetModulePackages(t, workspace, "libs/core/kernel")
	found, read := getOversizedTypes(pkgs, func(path string) (structLimit, bool) {
		return getStructLimit(strings.TrimPrefix(strings.TrimPrefix(path, load.CoreImportPath), "/"))
	})
	for _, one := range found {
		t.Errorf("%s: %s", one.Position, one.Name)
	}
	count, ok := getExportedMethodCount(pkgs, load.CoreImportPath+"/container", "Config")
	if !ok {
		t.Fatal("container.Config is not declared: the rule on its methods is checked against nothing")
	}
	if count > maxConfigMethods {
		t.Errorf("container.Config has %d exported methods, more than %d", count, maxConfigMethods)
	}

	if read == 0 {
		t.Fatalf("%d types of the adapters, the container and the use cases read: the rule is checked against nothing", read)
	}
}

func countStructFields(shape *types.Struct, visited map[*types.Struct]bool) int {
	if visited[shape] {
		return 0
	}
	visited[shape] = true
	count := 0
	for field := range shape.Fields() {
		if inner, ok := getEmbeddedStruct(field); ok {
			count += countStructFields(inner, visited)
			continue
		}
		count++
	}
	return count
}

func getEmbeddedStruct(field *types.Var) (*types.Struct, bool) {
	if !field.Embedded() {
		return nil, false
	}
	t := field.Type()
	if pointer, ok := types.Unalias(t).(*types.Pointer); ok {
		t = pointer.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return nil, false
	}
	shape, ok := named.Underlying().(*types.Struct)
	return shape, ok
}
