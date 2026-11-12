package deadexports

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// layout names where the check finds the protocol, the SDKs and the libraries whose uses pass on to
// their own consumers. The directories are relative to root and written with slashes.
type layout struct {
	root        string
	modulePath  string
	connectPath string
	sdkGoPath   string
	protocolDir string
	sdkTSDir    string
	libraryDirs []string
}

func newRepositoryLayout(t *testing.T) layout {
	t.Helper()
	return layout{
		root:        getRealDir(t, load.RepoRootDir),
		modulePath:  load.NumenRoot,
		connectPath: load.Wire + "/gen/numen/v1/numenv1connect",
		sdkGoPath:   load.SdkModule,
		protocolDir: "libs/core/protocol",
		sdkTSDir:    "libs/plugin/sdk-ts",
		libraryDirs: []string{"libs/plugin/sdk-go", "libs/plugin/sdk-ts", "libs/core/client"},
	}
}

func getRealDir(t *testing.T, dir string) string {
	t.Helper()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

// item is an RPC of the protocol or an export of an SDK, by its node key.
type item struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Position string `json:"position"`
}

// use is a reference to the node target from a file; user is the node of the declaration that holds
// the reference, which matters when the file is in a library that passes its uses on.
type use struct {
	target string
	file   string
	user   string
}

// usage is what one language reads: the items, the uses and the edges along which the consumers of
// the second node of a pair become consumers of the first.
type usage struct {
	items []item
	uses  []use
	edges [][2]string
}

// The node keys: "rpc:<Service>.<Method>", "go:<import path>|<Name>" with "<Type>.<Method>" for a
// method, "ts:<file>|<Name>" with "<Name>.<member>" for a member.
func getGoKey(pkgPath, name string) string {
	return "go:" + pkgPath + "|" + name
}

// getContainerKey is the type of a method or the declaration of a member, or "" for a top-level node.
func getContainerKey(key string) string {
	if !strings.HasPrefix(key, "go:") && !strings.HasPrefix(key, "ts:") {
		return ""
	}
	bar := strings.LastIndex(key, "|")
	if bar < 0 {
		return ""
	}
	dot := strings.Index(key[bar:], ".")
	if dot < 0 {
		return ""
	}
	return key[:bar+dot]
}

func getReceiverName(fn *types.Func) string {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return ""
	}
	recv := sig.Recv().Type()
	if pointer, ok := recv.(*types.Pointer); ok {
		recv = pointer.Elem()
	}
	if named, ok := types.Unalias(recv).(*types.Named); ok {
		return named.Origin().Obj().Name()
	}
	return ""
}

func isMethod(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	return ok && sig.Recv() != nil
}

func isExportedName(name string) bool {
	return name != "" && unicode.IsUpper([]rune(name)[0])
}

// getGoTargetKey is the node an object of a use stands for: an RPC of a generated client, or a
// declaration of the Go SDK at package level or a method; "" for anything else.
func (l layout) getGoTargetKey(obj types.Object) string {
	if obj == nil || obj.Pkg() == nil {
		return ""
	}
	pkgPath := obj.Pkg().Path()
	fn, isFunc := obj.(*types.Func)
	if isFunc {
		fn = fn.Origin()
	}
	if pkgPath == l.connectPath {
		if !isFunc {
			return ""
		}
		recv := getReceiverName(fn)
		if !isExportedName(recv) || !strings.HasSuffix(recv, "ServiceClient") {
			return ""
		}
		return "rpc:" + strings.TrimSuffix(recv, "Client") + "." + fn.Name()
	}
	if !load.IsWithin(pkgPath, l.sdkGoPath) {
		return ""
	}
	if isFunc && isMethod(fn) {
		recv := getReceiverName(fn)
		if recv == "" {
			return ""
		}
		return getGoKey(pkgPath, recv+"."+fn.Name())
	}
	if obj.Parent() != obj.Pkg().Scope() {
		return ""
	}
	switch obj.(type) {
	case *types.Func, *types.TypeName, *types.Var, *types.Const:
		return getGoKey(pkgPath, obj.Name())
	}
	return ""
}

func (l layout) getRelative(file string) string {
	if real, err := filepath.EvalSymlinks(file); err == nil {
		file = real
	}
	rel, err := filepath.Rel(l.root, file)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

func (l layout) getShortName(pkgPath string) string {
	return strings.TrimPrefix(pkgPath, l.modulePath)
}

// collectGoUsage reads the RPCs of the generated clients, the exports of the Go SDK, the uses of both
// in every package, and the edges from an interface method of the SDK, or of a package outside it, to
// the SDK methods that answer it. Each module is one load, whose types are not those of another.
func collectGoUsage(modules [][]*packages.Package, l layout) usage {
	var found usage
	seenItems := map[string]bool{}
	seenUses := map[use]bool{}
	seenPairs := map[string]bool{}
	for _, pkgs := range modules {
		l.collectModuleUsage(pkgs, &found, seenItems, seenUses, seenPairs)
	}
	return found
}

func (l layout) collectModuleUsage(pkgs []*packages.Package, found *usage, seenItems map[string]bool, seenUses map[use]bool, seenPairs map[string]bool) {
	var roots []*types.Package
	for _, pkg := range pkgs {
		if pkg.Types != nil {
			roots = append(roots, pkg.Types)
		}
	}
	sdkTypes := l.listSDKTypes(roots)
	for _, pkg := range pkgs {
		if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		for _, one := range l.listGoItems(pkg) {
			if !seenItems[one.Key] {
				seenItems[one.Key] = true
				found.items = append(found.items, one)
			}
		}
		for _, file := range pkg.Syntax {
			rel := l.getRelative(pkg.Fset.Position(file.Pos()).Filename)
			for _, one := range l.listGoUses(pkg, file, rel) {
				if !seenUses[one] {
					seenUses[one] = true
					found.uses = append(found.uses, one)
				}
			}
		}
		if load.IsWithin(pkg.PkgPath, l.sdkGoPath) {
			found.edges = append(found.edges, l.listDispatchEdges(pkg, seenPairs)...)
		} else if pkg.PkgPath != l.connectPath {
			found.edges = append(found.edges, l.listOwnInterfaceEdges(pkg, sdkTypes)...)
		}
	}
}

func (l layout) listGoItems(pkg *packages.Package) []item {
	scope := pkg.Types.Scope()
	var found []item
	if pkg.PkgPath == l.connectPath {
		for _, name := range scope.Names() {
			object, ok := scope.Lookup(name).(*types.TypeName)
			if !ok || !object.Exported() || !strings.HasSuffix(name, "ServiceClient") {
				continue
			}
			iface, ok := object.Type().Underlying().(*types.Interface)
			if !ok {
				continue
			}
			for method := range iface.Methods() {
				key := l.getGoTargetKey(method)
				if key == "" {
					continue
				}
				found = append(found, item{Key: key, Name: strings.TrimPrefix(key, "rpc:"), Position: l.getPosition(pkg.Fset, method.Pos())})
			}
		}
		return found
	}
	if !load.IsWithin(pkg.PkgPath, l.sdkGoPath) || slices.Contains(strings.Split(pkg.PkgPath, "/"), "internal") {
		return nil
	}
	for _, name := range scope.Names() {
		object := scope.Lookup(name)
		if !object.Exported() {
			continue
		}
		switch object.(type) {
		case *types.Func, *types.TypeName:
			found = append(found, item{
				Key:      getGoKey(pkg.PkgPath, name),
				Name:     l.getShortName(pkg.PkgPath) + "." + name,
				Position: l.getPosition(pkg.Fset, object.Pos()),
			})
		}
	}
	return found
}

func (l layout) getPosition(fset *token.FileSet, pos token.Pos) string {
	position := fset.Position(pos)
	return fmt.Sprintf("%s:%d", l.getRelative(position.Filename), position.Line)
}

// getGoDeclKey is the node of a declaration of a package, or "" for one that names no node.
func getGoDeclKey(pkg *packages.Package, name *ast.Ident) string {
	if name == nil || name.Name == "_" {
		return ""
	}
	object := pkg.TypesInfo.Defs[name]
	if fn, ok := object.(*types.Func); ok && isMethod(fn) {
		recv := getReceiverName(fn)
		if recv == "" {
			return ""
		}
		return getGoKey(pkg.PkgPath, recv+"."+name.Name)
	}
	return getGoKey(pkg.PkgPath, name.Name)
}

func (l layout) listGoUses(pkg *packages.Package, file *ast.File, rel string) []use {
	var found []use
	inspect := func(node ast.Node, user string) {
		ast.Inspect(node, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			object := pkg.TypesInfo.Uses[ident]
			target := l.getGoTargetKey(object)
			if target == "" {
				target = l.getOwnInterfaceMethodKey(object)
			}
			if target != "" {
				found = append(found, use{target: target, file: rel, user: user})
			}
			return true
		})
	}
	for _, decl := range file.Decls {
		switch current := decl.(type) {
		case *ast.FuncDecl:
			inspect(current, getGoDeclKey(pkg, current.Name))
		case *ast.GenDecl:
			if current.Tok == token.IMPORT {
				continue
			}
			for _, spec := range current.Specs {
				switch one := spec.(type) {
				case *ast.TypeSpec:
					inspect(one, getGoDeclKey(pkg, one.Name))
				case *ast.ValueSpec:
					user := ""
					for _, name := range one.Names {
						if user = getGoDeclKey(pkg, name); user != "" {
							break
						}
					}
					inspect(one, user)
				}
			}
		}
	}
	return found
}

// listDispatchEdges links each method of an interface of the SDK to the methods of the SDK types that
// answer it: a consumer that calls the interface method calls whichever type stands behind it.
func (l layout) listDispatchEdges(pkg *packages.Package, seenPairs map[string]bool) [][2]string {
	var named []*types.Named
	addScope := func(scope *types.Scope) {
		for _, name := range scope.Names() {
			object, ok := scope.Lookup(name).(*types.TypeName)
			if !ok || object.IsAlias() {
				continue
			}
			if one, ok := object.Type().(*types.Named); ok && one.TypeParams().Len() == 0 {
				named = append(named, one)
			}
		}
	}
	addScope(pkg.Types.Scope())
	for _, imported := range pkg.Types.Imports() {
		if load.IsWithin(imported.Path(), l.sdkGoPath) {
			addScope(imported.Scope())
		}
	}
	var edges [][2]string
	for _, ifaceType := range named {
		iface, ok := ifaceType.Underlying().(*types.Interface)
		if !ok || iface.NumMethods() == 0 {
			continue
		}
		for _, concrete := range named {
			if types.IsInterface(concrete) {
				continue
			}
			pair := concrete.Obj().Pkg().Path() + "." + concrete.Obj().Name() + "|" + ifaceType.Obj().Pkg().Path() + "." + ifaceType.Obj().Name()
			if seenPairs[pair] {
				continue
			}
			seenPairs[pair] = true
			pointer := types.NewPointer(concrete)
			if !types.Implements(concrete, iface) && !types.Implements(pointer, iface) {
				continue
			}
			for method := range iface.Methods() {
				user := l.getGoTargetKey(method)
				answer, _, _ := types.LookupFieldOrMethod(pointer, true, method.Pkg(), method.Name())
				fn, ok := answer.(*types.Func)
				if !ok || user == "" {
					continue
				}
				if target := l.getGoTargetKey(fn); target != "" && target != user {
					edges = append(edges, [2]string{target, user})
				}
			}
		}
	}
	return edges
}

// getOwnInterfaceMethodKey is the node of a method of an interface that a package of the repository
// outside the SDK and the protocol declares, such as a port of a plugin; "" for anything else.
func (l layout) getOwnInterfaceMethodKey(obj types.Object) string {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil || !isMethod(fn) {
		return ""
	}
	pkgPath := fn.Pkg().Path()
	if !strings.HasPrefix(pkgPath, l.modulePath) || pkgPath == l.connectPath || load.IsWithin(pkgPath, l.sdkGoPath) {
		return ""
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || !types.IsInterface(sig.Recv().Type()) {
		return ""
	}
	name := getReceiverName(fn)
	if name == "" {
		return ""
	}
	return getGoKey(pkgPath, name+"."+fn.Name())
}

// listSDKTypes lists the named types of the SDK packages among roots and the packages they import
// directly or through other packages, interfaces and generic types left out.
func (l layout) listSDKTypes(roots []*types.Package) []*types.Named {
	var named []*types.Named
	seen := map[string]bool{}
	var walk func(one *types.Package)
	walk = func(one *types.Package) {
		if seen[one.Path()] {
			return
		}
		seen[one.Path()] = true
		if load.IsWithin(one.Path(), l.sdkGoPath) {
			scope := one.Scope()
			for _, name := range scope.Names() {
				object, ok := scope.Lookup(name).(*types.TypeName)
				if !ok || object.IsAlias() {
					continue
				}
				if typ, ok := object.Type().(*types.Named); ok && typ.TypeParams().Len() == 0 && !types.IsInterface(typ) {
					named = append(named, typ)
				}
			}
		}
		for _, imported := range one.Imports() {
			walk(imported)
		}
	}
	for _, root := range roots {
		walk(root)
	}
	return named
}

// listOwnInterfaceEdges links each method of an interface the package declares, such as a port of a
// plugin, to the methods of the SDK types that answer it: a plugin that holds an SDK client behind a
// port of its own calls the client when it calls the port.
func (l layout) listOwnInterfaceEdges(pkg *packages.Package, sdkTypes []*types.Named) [][2]string {
	var interfaces []*types.Interface
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		object, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || object.IsAlias() {
			continue
		}
		typ, ok := object.Type().(*types.Named)
		if !ok || typ.TypeParams().Len() > 0 {
			continue
		}
		if iface, ok := typ.Underlying().(*types.Interface); ok && iface.NumMethods() > 0 {
			interfaces = append(interfaces, iface)
		}
	}
	if len(interfaces) == 0 {
		return nil
	}
	var edges [][2]string
	for _, iface := range interfaces {
		for _, concrete := range sdkTypes {
			pointer := types.NewPointer(concrete)
			if !types.Implements(concrete, iface) && !types.Implements(pointer, iface) {
				continue
			}
			for method := range iface.Methods() {
				user := l.getOwnInterfaceMethodKey(method)
				answer, _, _ := types.LookupFieldOrMethod(pointer, true, method.Pkg(), method.Name())
				fn, ok := answer.(*types.Func)
				if !ok || user == "" {
					continue
				}
				if target := l.getGoTargetKey(fn); target != "" {
					edges = append(edges, [2]string{target, user})
				}
			}
		}
	}
	return edges
}

// tsOutput is what uses.mjs prints.
type tsOutput struct {
	Items    []item      `json:"items"`
	Uses     [][3]string `json:"uses"`
	Packages int         `json:"packages"`
	Files    int         `json:"files"`
}

// collectTSUsage runs uses.mjs, beside this file, over the TypeScript and Vue sources of the tree.
func collectTSUsage(t *testing.T, l layout) (usage, tsOutput) {
	t.Helper()
	script := filepath.Join(getRealDir(t, load.RepoRootDir), "tools", "checks", "structure", "go", "dead-exports", "uses.mjs")
	args := append([]string{script, l.root, l.protocolDir, l.sdkTSDir}, l.libraryDirs...)
	command := exec.CommandContext(t.Context(), "node", args...)
	command.Stderr = os.Stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("node %s: %v", script, err)
	}
	var read tsOutput
	if err := json.Unmarshal(output, &read); err != nil {
		t.Fatalf("the output of %s: %v", script, err)
	}
	found := usage{items: read.Items}
	for _, one := range read.Uses {
		found.uses = append(found.uses, use{target: one[0], file: one[1], user: one[2]})
	}
	return found, read
}

type place int

const (
	placeNone place = iota
	placeLibrary
	placeConsumer
)

// isTestPath reports a test, a story, a test kit, a fixture or a template, which is no consumer.
func isTestPath(rel string) bool {
	for _, suffix := range []string{"_test.go", ".test.ts", ".spec.ts", ".stories.ts", ".test.mjs", ".d.ts"} {
		if strings.HasSuffix(rel, suffix) {
			return true
		}
	}
	segments := strings.Split(rel, "/")
	for _, segment := range segments[:len(segments)-1] {
		if strings.HasPrefix(segment, ".") || strings.HasSuffix(segment, "test") || slices.Contains([]string{"testdata", "testing", "fixtures", "e2e", "node_modules"}, segment) {
			return true
		}
	}
	return false
}

func isWithinDir(rel, dir string) bool {
	return rel == dir || strings.HasPrefix(rel, dir+"/")
}

// classify tells whether a use in the file rel counts, passes on, or names a consumer, and which one:
// a plugin by its folder, an application by its folder, any other package by its directory.
func (l layout) classify(rel string) (place, string) {
	if rel == "" || isTestPath(rel) || isWithinDir(rel, l.protocolDir) {
		return placeNone, ""
	}
	for _, dir := range l.libraryDirs {
		if isWithinDir(rel, dir) {
			return placeLibrary, ""
		}
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 3 {
		return placeNone, ""
	}
	switch parts[0] {
	case "plugins":
		return placeConsumer, "plugin " + parts[1]
	case "apps":
		return placeConsumer, "app " + parts[1]
	case "libs":
		return placeConsumer, "package " + path.Dir(rel)
	}
	return placeNone, ""
}

func getItemFile(one item) string {
	file, _, _ := strings.Cut(one.Position, ":")
	return file
}

// finding is an item that nothing outside tests uses.
type finding struct {
	name     string
	position string
}

func (f finding) String() string {
	return f.position + ": " + f.name + ": used by nothing outside tests: dead"
}

// getConsumers propagates the consumers along the edges until nothing changes.
func getConsumers(parts []usage, l layout) map[string]map[string]bool {
	consumers := map[string]map[string]bool{}
	users := map[string]map[string]bool{}
	addEdge := func(target, user string) {
		if target == "" || user == "" || target == user {
			return
		}
		if users[target] == nil {
			users[target] = map[string]bool{}
		}
		users[target][user] = true
	}
	addContainer := func(key string) {
		if container := getContainerKey(key); container != "" {
			addEdge(container, key)
		}
	}
	for _, part := range parts {
		for _, one := range part.uses {
			addContainer(one.target)
			addContainer(one.user)
			switch kind, consumer := l.classify(one.file); kind {
			case placeLibrary:
				addEdge(one.target, one.user)
			case placeConsumer:
				if consumers[one.target] == nil {
					consumers[one.target] = map[string]bool{}
				}
				consumers[one.target][consumer] = true
			case placeNone:
			}
		}
		for _, edge := range part.edges {
			addContainer(edge[0])
			addContainer(edge[1])
			addEdge(edge[0], edge[1])
		}
	}
	for changed := true; changed; {
		changed = false
		for target, from := range users {
			for user := range from {
				for consumer := range consumers[user] {
					if consumers[target][consumer] {
						continue
					}
					if consumers[target] == nil {
						consumers[target] = map[string]bool{}
					}
					consumers[target][consumer] = true
					changed = true
				}
			}
		}
	}
	return consumers
}

// isGeneratedContract reports a file generated from a contract of an SDK: a `*.pb.go` or `*_pb.ts`
// file under a `contracts` directory, whose declarations protoc writes for the whole message set.
func isGeneratedContract(rel string) bool {
	if !strings.HasSuffix(rel, ".pb.go") && !strings.HasSuffix(rel, "_pb.ts") {
		return false
	}
	return slices.Contains(strings.Split(path.Dir(rel), "/"), "contracts")
}

// findDeadExports lists every item that nothing outside tests uses.
func findDeadExports(parts []usage, l layout) []finding {
	consumers := getConsumers(parts, l)
	var found []finding
	seen := map[string]bool{}
	for _, part := range parts {
		for _, one := range part.items {
			file := getItemFile(one)
			if seen[one.Key] || (!strings.HasPrefix(one.Key, "rpc:") && (isTestPath(file) || isGeneratedContract(file))) {
				continue
			}
			seen[one.Key] = true
			if len(consumers[one.Key]) == 0 {
				found = append(found, finding{name: one.Name, position: one.Position})
			}
		}
	}
	slices.SortFunc(found, func(a, b finding) int { return strings.Compare(a.name, b.name) })
	return found
}

func checkDeadExports(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	l := newRepositoryLayout(t)
	var modules [][]*packages.Package
	for _, dir := range load.ListWorkspaceModules(t) {
		modules = append(modules, load.ListProductionPackages(workspace.ListModule(t, dir)))
	}
	goUsage := collectGoUsage(modules, l)
	tsUsage, read := collectTSUsage(t, l)
	counts := map[string]int{}
	for _, one := range append(slices.Clone(goUsage.items), tsUsage.items...) {
		kind, _, _ := strings.Cut(one.Key, ":")
		counts[kind]++
	}
	rpcUses := 0
	for _, one := range append(slices.Clone(goUsage.uses), tsUsage.uses...) {
		if strings.HasPrefix(one.target, "rpc:") {
			rpcUses++
		}
	}
	if counts["rpc"] == 0 || counts["go"] == 0 || counts["ts"] == 0 || rpcUses == 0 || read.Packages == 0 || read.Files == 0 {
		t.Fatalf("%v items, %d uses of an RPC, %d TypeScript packages and %d files: the rule is checked against nothing", counts, rpcUses, read.Packages, read.Files)
	}
	for _, one := range findDeadExports([]usage{goUsage, tsUsage}, l) {
		t.Error(one)
	}
}
