package jsonkeys

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// jsonKeyViolation is a struct field whose JSON key breaks the camelCase rule.
type jsonKeyViolation struct {
	position string
	detail   string
}

func isCamelCaseKey(key string) bool {
	if key == "" || key[0] < 'a' || key[0] > 'z' {
		return false
	}
	for i := 1; i < len(key); i++ {
		if !isCamelCaseTailByte(key[i]) {
			return false
		}
	}
	return true
}

func isCamelCaseTailByte(ch byte) bool {
	isLower := ch >= 'a' && ch <= 'z'
	isUpper := ch >= 'A' && ch <= 'Z'
	isDigit := ch >= '0' && ch <= '9'
	return isLower || isUpper || isDigit
}

func parseJSONTagName(rawTag string) (name string, hasTag bool) {
	value, ok := reflect.StructTag(rawTag).Lookup("json")
	if !ok {
		return "", false
	}
	head, _, _ := strings.Cut(value, ",")
	return head, true
}

func importsExternalWire(pkg *packages.Package) bool {
	_, hasExec := pkg.Imports["os/exec"]
	_, hasHTTP := pkg.Imports["net/http"]
	return hasExec || hasHTTP
}

func isCheckedNamedType(named *types.Named) bool {
	pkg := named.Origin().Obj().Pkg()
	if pkg == nil {
		return false
	}
	path := pkg.Path()
	if strings.HasPrefix(path, load.Wire) {
		return false
	}
	return strings.HasPrefix(path, "github.com/jiva-studio/numen/") || strings.HasPrefix(path, "fixture")
}

func hasCustomJSONMarshal(named *types.Named) bool {
	methods := types.NewMethodSet(types.NewPointer(named.Origin()))
	for method := range methods.Methods() {
		name := method.Obj().Name()
		if name == "MarshalJSON" || name == "MarshalText" {
			return true
		}
	}
	return false
}

func hasAnyJSONTag(st *types.Struct) bool {
	for i := range st.NumFields() {
		if _, ok := parseJSONTagName(st.Tag(i)); ok {
			return true
		}
	}
	return false
}

func listJSONBoundaryGenerics() []string {
	return []string{
		load.SdkModule + "/cache.New",
		load.SdkModule + "/server.Handle",
		load.SdkModule + "/server.Stream",
		load.SdkModule + "/tools.Add",
		"fixture/sdk.Handle",
		"fixture/sdk.Cache",
	}
}

func isJSONBoundaryGeneric(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}
	return slices.Contains(listJSONBoundaryGenerics(), fn.Pkg().Path()+"."+fn.Name())
}

func isMarshalIndentCall(call *ast.CallExpr, info *types.Info) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	fn, ok := info.Uses[sel.Sel].(*types.Func)
	return ok && fn.Pkg() != nil && fn.Pkg().Path() == "encoding/json" && fn.Name() == "MarshalIndent"
}

// jsonKeyWalk collects non-camelCase JSON struct tags and untagged boundary fields.
type jsonKeyWalk struct {
	fset         *token.FileSet
	visitedTypes map[types.Type]bool
	found        map[jsonKeyViolation]bool
	checkedTags  int
}

func (w *jsonKeyWalk) record(pos token.Pos, detail string) {
	position := w.fset.Position(pos)
	loc := filepath.ToSlash(position.Filename) + ":" + strconv.Itoa(position.Line)
	w.found[jsonKeyViolation{position: loc, detail: detail}] = true
}

func (w *jsonKeyWalk) checkStructFields(st *types.Struct, requireTags bool) {
	for i := range st.NumFields() {
		field := st.Field(i)
		name, hasTag := parseJSONTagName(st.Tag(i))
		if hasTag {
			w.checkedTags++
		}
		if hasTag && name != "-" && !isCamelCaseKey(name) {
			w.record(field.Pos(), field.Name()+" has non-camelCase json tag "+strconv.Quote(name))
		}
		if requireTags && !hasTag && field.Exported() && !field.Embedded() {
			w.record(field.Pos(), field.Name()+" has no json tag")
		}
		if name != "-" && field.Exported() {
			w.walkType(field.Type(), requireTags)
		}
	}
}

func (w *jsonKeyWalk) walkNamed(named *types.Named, requireTags bool) {
	if !isCheckedNamedType(named) || w.visitedTypes[named] {
		return
	}
	w.visitedTypes[named] = true
	st, ok := named.Origin().Underlying().(*types.Struct)
	if !ok {
		w.walkType(named.Origin().Underlying(), requireTags)
		return
	}
	custom := hasCustomJSONMarshal(named)
	if requireTags || hasAnyJSONTag(st) {
		w.checkStructFields(st, requireTags && !custom)
	}
}

func (w *jsonKeyWalk) walkType(t types.Type, requireTags bool) {
	switch current := types.Unalias(t).(type) {
	case *types.Named:
		w.walkNamed(current, requireTags)
	case *types.Pointer:
		w.walkType(current.Elem(), requireTags)
	case *types.Slice:
		w.walkType(current.Elem(), requireTags)
	case *types.Array:
		w.walkType(current.Elem(), requireTags)
	case *types.Map:
		w.walkType(current.Elem(), requireTags)
	case *types.Struct:
		if requireTags || hasAnyJSONTag(current) {
			w.checkStructFields(current, requireTags)
		}
	}
}

func (w *jsonKeyWalk) inspectStructTree(root ast.Node, info *types.Info, customMarshal bool) {
	ast.Inspect(root, func(node ast.Node) bool {
		structNode, ok := node.(*ast.StructType)
		if !ok {
			return true
		}
		tv, ok := info.Types[structNode]
		if !ok {
			return true
		}
		if st, ok := tv.Type.(*types.Struct); ok && hasAnyJSONTag(st) {
			w.checkStructFields(st, !customMarshal)
		}
		return true
	})
}

func (w *jsonKeyWalk) inspectFile(pkg *packages.Package, file *ast.File) {
	externalWire := importsExternalWire(pkg)
	for _, decl := range file.Decls {
		switch current := decl.(type) {
		case *ast.GenDecl:
			w.inspectGenDecl(current, pkg.TypesInfo, externalWire)
		case *ast.FuncDecl:
			if !externalWire && current.Body != nil {
				w.inspectStructTree(current.Body, pkg.TypesInfo, false)
			}
		}
	}
	w.inspectBoundaryCalls(pkg, file)
}

func (w *jsonKeyWalk) inspectGenDecl(gen *ast.GenDecl, info *types.Info, externalWire bool) {
	for _, spec := range gen.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			if !externalWire {
				w.inspectStructTree(spec, info, false)
			}
			continue
		}
		if !ast.IsExported(typeSpec.Name.Name) && externalWire {
			continue
		}
		named, isNamed := info.Defs[typeSpec.Name].Type().(*types.Named)
		custom := isNamed && hasCustomJSONMarshal(named)
		w.inspectStructTree(typeSpec.Type, info, custom)
	}
}

func (w *jsonKeyWalk) inspectBoundaryCalls(pkg *packages.Package, file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.Ident:
			instance, ok := pkg.TypesInfo.Instances[current]
			if ok && isJSONBoundaryGeneric(pkg.TypesInfo.Uses[current]) {
				for arg := range instance.TypeArgs.Types() {
					w.walkType(arg, true)
				}
			}
		case *ast.CallExpr:
			if isMarshalIndentCall(current, pkg.TypesInfo) && len(current.Args) > 0 {
				w.walkType(pkg.TypesInfo.TypeOf(current.Args[0]), true)
			}
		}
		return true
	})
}

func getJSONKeyViolations(pkgs []*packages.Package) ([]jsonKeyViolation, int) {
	var violations []jsonKeyViolation
	visited := map[jsonKeyViolation]bool{}
	checked := 0
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil || strings.HasPrefix(pkg.PkgPath, load.Wire) {
			continue
		}
		walk := &jsonKeyWalk{
			fset:         pkg.Fset,
			visitedTypes: map[types.Type]bool{},
			found:        map[jsonKeyViolation]bool{},
		}
		for _, file := range pkg.Syntax {
			if ast.IsGenerated(file) {
				continue
			}
			walk.inspectFile(pkg, file)
		}
		checked += walk.checkedTags
		for one := range walk.found {
			if !visited[one] {
				visited[one] = true
				violations = append(violations, one)
			}
		}
	}
	slices.SortFunc(violations, func(a, b jsonKeyViolation) int {
		if cmp := strings.Compare(a.position, b.position); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.detail, b.detail)
	})
	return violations, checked
}

func checkProjectJSONKeysAreCamelCase(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	var pkgs []*packages.Package
	for _, dir := range load.ListWorkspaceModules(t) {
		pkgs = append(pkgs, workspace.ListModule(t, dir)...)
	}
	violations, checked := getJSONKeyViolations(pkgs)
	for _, one := range violations {
		t.Errorf("%s: %s", one.position, one.detail)
	}
	if checked < 200 {
		t.Fatalf("%d JSON struct tags checked: the rule is checked against nothing", checked)
	}
}
