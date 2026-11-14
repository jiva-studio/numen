package apitypes

import (
	"encoding/json"
	"go/ast"
	"go/constant"
	"go/types"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// The kinds of a shape: the JSON a value is sent or read as.
const (
	kindString  = "string"
	kindNumber  = "number"
	kindBoolean = "boolean"
	kindAny     = "any"
	kindArray   = "array"
	kindRecord  = "record"
	kindObject  = "object"
	kindOther   = "other"
)

// maxDepth is how many functions a method name may pass through before the call that names it.
const maxDepth = 6

// shape is the JSON one side sends or reads for a value, in the form types.mjs prints it. Values are the
// strings an enumeration allows, Text describes a type of the kind other.
type shape struct {
	Kind    string           `json:"kind"`
	Values  []string         `json:"values,omitempty"`
	Fields  map[string]field `json:"fields,omitempty"`
	Element *shape           `json:"element,omitempty"`
	Text    string           `json:"text,omitempty"`
}

type field struct {
	IsOptional bool  `json:"isOptional"`
	Shape      shape `json:"shape"`
}

// tsCall is one call of a JSON method that a plugin frontend makes.
type tsCall struct {
	Method   string `json:"method"`
	Place    string `json:"place"`
	Request  shape  `json:"request"`
	Response shape  `json:"response"`
}

// tsFailure is a call of the Api whose method or types the type checker cannot name.
type tsFailure struct {
	Place  string `json:"place"`
	Reason string `json:"reason"`
}

// frontend is what types.mjs prints for one plugin.
type frontend struct {
	Files    int         `json:"files"`
	Calls    []tsCall    `json:"calls"`
	Failures []tsFailure `json:"failures"`
}

// servedMethod is a JSON method a plugin backend serves, with what its request and response send.
type servedMethod struct {
	name     string
	request  shape
	response shape
}

// finding is one difference, at a place relative to the repository root.
type finding struct {
	place string
	text  string
}

// layout names the import path of the repository and the package of the SDK that serves JSON methods.
type layout struct {
	root       string
	modulePath string
	serverPath string
}

func newRepositoryLayout(t *testing.T) layout {
	t.Helper()
	return layout{
		root:       getRealDir(t, load.RepoRootDir),
		modulePath: load.NumenRoot,
		serverPath: load.SdkModule + "/server",
	}
}

func getRealDir(t *testing.T, dir string) string {
	t.Helper()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// getPluginName is the folder under plugins/ that holds a package, or "" for a package of no plugin.
func (l layout) getPluginName(pkgPath string) string {
	rest, ok := strings.CutPrefix(pkgPath, l.modulePath+"plugins/")
	if !ok {
		return ""
	}
	name, _, _ := strings.Cut(rest, "/")
	return name
}

// goSite is a call of a function, with the package whose type information covers it.
type goSite struct {
	pkg   *packages.Package
	call  *ast.CallExpr
	ident *ast.Ident
}

// goParameter is a parameter of a declared function, by its index.
type goParameter struct {
	fn    *types.Func
	index int
}

// goMethod is a method name a call stands for, with the type parameters bound on its way.
type goMethod struct {
	name     string
	position string
	bindings map[*types.TypeParam]types.Type
	reason   string
}

func getCalleeIdent(expr ast.Expr) *ast.Ident {
	switch one := ast.Unparen(expr).(type) {
	case *ast.Ident:
		return one
	case *ast.SelectorExpr:
		return one.Sel
	case *ast.IndexExpr:
		return getCalleeIdent(one.X)
	case *ast.IndexListExpr:
		return getCalleeIdent(one.X)
	}
	return nil
}

// goIndex is the calls and the parameters of the functions of the plugin packages.
type goIndex struct {
	callers    map[*types.Func][]goSite
	parameters map[*types.Var]goParameter
}

func newGoIndex(pkgs []*packages.Package) goIndex {
	index := goIndex{callers: map[*types.Func][]goSite{}, parameters: map[*types.Var]goParameter{}}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				switch one := node.(type) {
				case *ast.FuncDecl:
					if fn, ok := pkg.TypesInfo.Defs[one.Name].(*types.Func); ok {
						params := fn.Signature().Params()
						for i := range params.Len() {
							index.parameters[params.At(i)] = goParameter{fn: fn, index: i}
						}
					}
				case *ast.CallExpr:
					ident := getCalleeIdent(one.Fun)
					if ident == nil {
						return true
					}
					if fn, ok := pkg.TypesInfo.Uses[ident].(*types.Func); ok {
						origin := fn.Origin()
						index.callers[origin] = append(index.callers[origin], goSite{pkg: pkg, call: one, ident: ident})
					}
				}
				return true
			})
		}
	}
	return index
}

func getPosition(pkg *packages.Package, node ast.Node, root string) string {
	position := pkg.Fset.Position(node.Pos())
	rel, err := filepath.Rel(root, position.Filename)
	if err != nil {
		rel = position.Filename
	}
	return filepath.ToSlash(rel) + ":" + strconv.Itoa(position.Line)
}

// getMethods follows a method argument up through the functions that pass it on to the calls that name it
// by a constant.
func (x goIndex) getMethods(site goSite, expr ast.Expr, bindings map[*types.TypeParam]types.Type, depth int, root string) []goMethod {
	if value := site.pkg.TypesInfo.Types[expr].Value; value != nil && value.Kind() == constant.String {
		return []goMethod{{name: constant.StringVal(value), position: getPosition(site.pkg, expr, root), bindings: bindings, reason: ""}}
	}
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	variable, isVar := site.pkg.TypesInfo.Uses[ident].(*types.Var)
	parameter, isParameter := x.parameters[variable]
	if !ok || !isVar || !isParameter {
		return []goMethod{{name: "", position: getPosition(site.pkg, expr, root), bindings: nil, reason: "the method is not a constant string"}}
	}
	if depth >= maxDepth {
		return []goMethod{{name: "", position: getPosition(site.pkg, expr, root), bindings: nil, reason: "the method passes through too many functions"}}
	}
	var methods []goMethod
	typeParams := parameter.fn.Signature().TypeParams()
	for _, outer := range x.callers[parameter.fn] {
		next := make(map[*types.TypeParam]types.Type, len(bindings)+typeParams.Len())
		maps.Copy(next, bindings)
		instance := outer.pkg.TypesInfo.Instances[outer.ident]
		for i := range typeParams.Len() {
			if instance.TypeArgs != nil && i < instance.TypeArgs.Len() {
				next[typeParams.At(i)] = instance.TypeArgs.At(i)
			}
		}
		if parameter.index >= len(outer.call.Args) {
			continue
		}
		methods = append(methods, x.getMethods(outer, outer.call.Args[parameter.index], next, depth+1, root)...)
	}
	return methods
}

// listServedMethods reads the JSON methods each plugin serves through Handle and Stream of the SDK server,
// by plugin name.
func listServedMethods(pkgs []*packages.Package, l layout) (map[string][]servedMethod, []finding) {
	index := newGoIndex(pkgs)
	served := map[string][]servedMethod{}
	var findings []finding
	for fn, sites := range index.callers {
		if fn.Pkg() == nil || fn.Pkg().Path() != l.serverPath || (fn.Name() != "Handle" && fn.Name() != "Stream") {
			continue
		}
		for _, site := range sites {
			plugin := l.getPluginName(site.pkg.PkgPath)
			instance := site.pkg.TypesInfo.Instances[site.ident]
			if plugin == "" || len(site.call.Args) < 2 || instance.TypeArgs == nil || instance.TypeArgs.Len() != 2 {
				continue
			}
			for _, method := range index.getMethods(site, site.call.Args[1], nil, 0, l.root) {
				if method.reason != "" {
					findings = append(findings, finding{place: method.position, text: "the backend serves a method the check cannot name: " + method.reason})
					continue
				}
				builder := shapeBuilder{bindings: method.bindings}
				served[plugin] = append(served[plugin], servedMethod{
					name:     method.name,
					request:  builder.getShape(instance.TypeArgs.At(0), nil),
					response: builder.getShape(instance.TypeArgs.At(1), nil),
				})
			}
		}
	}
	return served, findings
}

// shapeBuilder describes Go types as encoding/json writes and reads them.
type shapeBuilder struct {
	bindings map[*types.TypeParam]types.Type
}

func hasMethod(t types.Type, name string) bool {
	object, _, _ := types.LookupFieldOrMethod(types.NewPointer(t), true, nil, name)
	_, ok := object.(*types.Func)
	return ok
}

// listEnumValues is the non-empty string constants of a named type, sorted.
func listEnumValues(named *types.Named) []string {
	pkg := named.Obj().Pkg()
	if pkg == nil {
		return nil
	}
	var values []string
	for _, name := range pkg.Scope().Names() {
		value, ok := pkg.Scope().Lookup(name).(*types.Const)
		if !ok || !types.Identical(value.Type(), named) || value.Val().Kind() != constant.String {
			continue
		}
		if text := constant.StringVal(value.Val()); text != "" {
			values = append(values, text)
		}
	}
	slices.Sort(values)
	return slices.Compact(values)
}

func (b shapeBuilder) getShape(t types.Type, stack []types.Type) shape {
	t = types.Unalias(t)
	switch one := t.(type) {
	case *types.TypeParam:
		bound, ok := b.bindings[one]
		if !ok || len(stack) > 64 {
			return shape{Kind: kindOther, Text: one.String()}
		}
		return b.getShape(bound, append(stack, one))
	case *types.Pointer:
		return b.getShape(one.Elem(), stack)
	case *types.Named:
		if slices.ContainsFunc(stack, func(seen types.Type) bool { return types.Identical(seen, one) }) || hasMethod(one, "MarshalJSON") {
			return shape{Kind: kindAny}
		}
		if hasMethod(one, "MarshalText") {
			return shape{Kind: kindString}
		}
		if basic, ok := one.Underlying().(*types.Basic); ok && basic.Info()&types.IsString != 0 {
			return shape{Kind: kindString, Values: listEnumValues(one)}
		}
		stack = append(stack, one)
	}
	switch one := t.Underlying().(type) {
	case *types.Basic:
		return getBasicShape(one)
	case *types.Slice:
		return b.getListShape(one.Elem(), stack)
	case *types.Array:
		return b.getListShape(one.Elem(), stack)
	case *types.Map:
		element := b.getShape(one.Elem(), stack)
		return shape{Kind: kindRecord, Element: &element}
	case *types.Interface:
		return shape{Kind: kindAny}
	case *types.Struct:
		return shape{Kind: kindObject, Fields: b.getFields(one, stack)}
	}
	return shape{Kind: kindOther, Text: t.String()}
}

func getBasicShape(basic *types.Basic) shape {
	info := basic.Info()
	switch {
	case info&types.IsString != 0:
		return shape{Kind: kindString}
	case info&types.IsBoolean != 0:
		return shape{Kind: kindBoolean}
	case info&types.IsNumeric != 0 && info&types.IsComplex == 0:
		return shape{Kind: kindNumber}
	}
	return shape{Kind: kindOther, Text: basic.String()}
}

// getListShape is a list of element, except bytes, which encoding/json sends as base64 text.
func (b shapeBuilder) getListShape(element types.Type, stack []types.Type) shape {
	if basic, ok := types.Unalias(element).(*types.Basic); ok && basic.Kind() == types.Byte {
		return shape{Kind: kindString}
	}
	inner := b.getShape(element, stack)
	return shape{Kind: kindArray, Element: &inner}
}

// readJSONTag is the name and the options of the json key of a struct tag, as reflect reads it, and
// whether the key leaves the field out.
func readJSONTag(tag string) (name string, options []string, isSkipped bool) {
	value, ok := reflect.StructTag(tag).Lookup("json")
	if !ok {
		return "", nil, false
	}
	if value == "-" {
		return "", nil, true
	}
	name, rest, _ := strings.Cut(value, ",")
	return name, strings.Split(rest, ","), false
}

// getFields is the JSON fields of a struct: its own first, then those of embedded structs without a json
// name that no shallower field shadows.
func (b shapeBuilder) getFields(st *types.Struct, stack []types.Type) map[string]field {
	fields := map[string]field{}
	var embedded []*types.Struct
	for i := range st.NumFields() {
		one := st.Field(i)
		name, options, isSkipped := readJSONTag(st.Tag(i))
		if isSkipped {
			continue
		}
		if one.Embedded() && name == "" {
			if inner, ok := types.Unalias(dereference(one.Type())).Underlying().(*types.Struct); ok {
				embedded = append(embedded, inner)
				continue
			}
		}
		if !one.Exported() {
			continue
		}
		if name == "" {
			name = one.Name()
		}
		_, isPointer := types.Unalias(one.Type()).(*types.Pointer)
		value := b.getShape(one.Type(), stack)
		if slices.Contains(options, "string") && (value.Kind == kindNumber || value.Kind == kindBoolean) {
			value = shape{Kind: kindString}
		}
		fields[name] = field{
			IsOptional: isPointer || slices.Contains(options, "omitempty") || slices.Contains(options, "omitzero"),
			Shape:      value,
		}
	}
	for _, inner := range embedded {
		for name, one := range b.getFields(inner, stack) {
			if _, ok := fields[name]; !ok {
				fields[name] = one
			}
		}
	}
	return fields
}

func dereference(t types.Type) types.Type {
	if pointer, ok := types.Unalias(t).(*types.Pointer); ok {
		return pointer.Elem()
	}
	return t
}

func describeKind(s shape) string {
	switch s.Kind {
	case kindOther:
		return s.Text
	case kindArray:
		return "an array"
	case kindObject:
		return "an object"
	}
	return "a " + s.Kind
}

// compareShapes adds a difference for each place where what Go sends or reads and what TypeScript reads
// or sends disagree. In a request a field may be optional on either side: a field left out is its zero
// value.
func compareShapes(findings *[]string, at string, goShape, tsShape shape, isRequest bool) {
	if goShape.Kind == kindAny || tsShape.Kind == kindAny {
		return
	}
	if goShape.Kind == kindOther || tsShape.Kind == kindOther || goShape.Kind != tsShape.Kind {
		*findings = append(*findings, at+": Go has "+describeKind(goShape)+", TypeScript has "+describeKind(tsShape))
		return
	}
	switch goShape.Kind {
	case kindString:
		declared := slices.Clone(tsShape.Values)
		slices.Sort(declared)
		declared = slices.DeleteFunc(slices.Compact(declared), func(value string) bool { return value == "" })
		switch {
		case len(goShape.Values) > 0 && len(declared) > 0:
			addDifferences(findings, at, "value", goShape.Values, declared)
		case len(goShape.Values) > 0:
			*findings = append(*findings, at+": Go has an enumeration ("+strings.Join(goShape.Values, ", ")+"), TypeScript has a plain string")
		case len(declared) > 0:
			*findings = append(*findings, at+": TypeScript has an enumeration ("+strings.Join(declared, ", ")+"), Go has a plain string")
		}
	case kindArray, kindRecord:
		if goShape.Element != nil && tsShape.Element != nil {
			compareShapes(findings, at+"[]", *goShape.Element, *tsShape.Element, isRequest)
		}
	case kindObject:
		compareFields(findings, at, goShape.Fields, tsShape.Fields, isRequest)
	}
}

func compareFields(findings *[]string, at string, goFields, tsFields map[string]field, isRequest bool) {
	addDifferences(findings, at, "field", getSortedKeys(goFields), getSortedKeys(tsFields))
	for _, name := range getSortedKeys(goFields) {
		goField := goFields[name]
		tsField, ok := tsFields[name]
		if !ok {
			continue
		}
		if !isRequest && goField.IsOptional != tsField.IsOptional {
			if goField.IsOptional {
				*findings = append(*findings, at+"."+name+": Go omits it when empty, TypeScript says it is always there")
			} else {
				*findings = append(*findings, at+"."+name+": Go always sends it, TypeScript says it may be absent")
			}
		}
		compareShapes(findings, at+"."+name, goField.Shape, tsField.Shape, isRequest)
	}
}

func addDifferences(findings *[]string, at, noun string, goNames, tsNames []string) {
	for _, name := range goNames {
		if !slices.Contains(tsNames, name) {
			*findings = append(*findings, at+": "+noun+" "+name+" is missing in TypeScript")
		}
	}
	for _, name := range tsNames {
		if !slices.Contains(goNames, name) {
			*findings = append(*findings, at+": "+noun+" "+name+" is not in Go")
		}
	}
}

func getSortedKeys(fields map[string]field) []string {
	return slices.Sorted(maps.Keys(fields))
}

// comparison is what one comparison of the plugins read.
type comparison struct {
	methods int
	calls   int
}

// compareAPIs compares, for each plugin that serves JSON methods, every call its frontend makes with the
// method the backend serves under that name. A plugin that serves methods and whose frontend calls none
// of them through the Api of the SDK has no TypeScript side, and fails.
func compareAPIs(served map[string][]servedMethod, frontends map[string]frontend) ([]finding, comparison) {
	var findings []finding
	var count comparison
	for _, plugin := range slices.Sorted(maps.Keys(served)) {
		dir := "plugins/" + plugin
		methods := served[plugin]
		count.methods += len(methods)
		side := frontends[plugin]
		for _, one := range side.Failures {
			findings = append(findings, finding{place: dir + "/" + one.Place, text: "the frontend calls a method whose name or types the type checker cannot read: " + one.Reason})
		}
		if len(side.Calls) == 0 && len(side.Failures) == 0 {
			findings = append(findings, finding{place: dir, text: "the backend serves " + strconv.Itoa(len(methods)) + " JSON method(s) and the frontend calls none of them through the Api of @numen/plugin-sdk"})
		}
		for _, call := range side.Calls {
			count.calls++
			matching := slices.DeleteFunc(slices.Clone(methods), func(one servedMethod) bool { return one.name != call.Method })
			if len(matching) == 0 {
				findings = append(findings, finding{place: dir + "/" + call.Place, text: "the frontend calls " + call.Method + ", which the backend does not serve"})
				continue
			}
			for _, method := range matching {
				var differences []string
				compareShapes(&differences, call.Method+" request", method.request, call.Request, true)
				compareShapes(&differences, call.Method+" response", method.response, call.Response, false)
				for _, text := range differences {
					findings = append(findings, finding{place: dir + "/" + call.Place, text: text})
				}
			}
		}
	}
	return findings, count
}

// readFrontends runs types.mjs on the plugin directories and returns its output by plugin name.
func readFrontends(t *testing.T, root string, dirs map[string]string) map[string]frontend {
	t.Helper()
	script := filepath.Join(getRealDir(t, load.RepoRootDir), "tools", "checks", "structure", "go", "api-types", "types.mjs")
	args := []string{script}
	for _, name := range slices.Sorted(maps.Keys(dirs)) {
		args = append(args, dirs[name])
	}
	command := exec.CommandContext(t.Context(), "node", args...)
	command.Dir = root
	command.Stderr = os.Stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("node %s: %v", script, err)
	}
	var byDir map[string]frontend
	if err := json.Unmarshal(output, &byDir); err != nil {
		t.Fatalf("the output of %s: %v", script, err)
	}
	byName := make(map[string]frontend, len(dirs))
	for name, dir := range dirs {
		byName[name] = byDir[dir]
	}
	return byName
}

// runAPITypes reads the served methods of pkgs and the calls of the frontends of their plugins and
// compares them.
func runAPITypes(t *testing.T, pkgs []*packages.Package, l layout) ([]finding, comparison) {
	t.Helper()
	served, findings := listServedMethods(pkgs, l)
	dirs := make(map[string]string, len(served))
	for plugin := range served {
		dirs[plugin] = filepath.Join(l.root, "plugins", plugin)
	}
	compared, count := compareAPIs(served, readFrontends(t, l.root, dirs))
	findings = append(findings, compared...)
	slices.SortFunc(findings, func(a, b finding) int {
		if order := strings.Compare(a.place, b.place); order != 0 {
			return order
		}
		return strings.Compare(a.text, b.text)
	})
	return findings, count
}

func checkPluginAPITypes(t *testing.T) {
	t.Helper()
	l := newRepositoryLayout(t)
	root, err := filepath.Abs(load.RepoRootDir)
	if err != nil {
		t.Fatal(err)
	}
	var modules []string
	for _, dir := range load.ListWorkspaceModules(t) {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(filepath.ToSlash(rel), "plugins/") {
			modules = append(modules, rel)
		}
	}
	workspace := load.Workspace(t, modules...)
	var pkgs []*packages.Package
	for _, rel := range modules {
		pkgs = append(pkgs, load.GetModulePackages(t, workspace, rel)...)
	}
	findings, count := runAPITypes(t, pkgs, l)
	for _, one := range findings {
		t.Errorf("%s: %s", one.place, one.text)
	}
	if count.methods == 0 || count.calls == 0 {
		t.Fatalf("%d served methods and %d frontend calls read: the rule is checked against nothing", count.methods, count.calls)
	}
}
