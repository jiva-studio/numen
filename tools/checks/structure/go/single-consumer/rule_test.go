package singleconsumer

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

// item is an RPC of the protocol or an export of an SDK, by its node key. The node keys are
// "rpc:<Service>.<Method>", "go:<import path>|<Name>" with "<Type>.<Method>" for a method, and
// "ts:<file>|<Name>" with "<Name>.<member>" for a member.
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
// the second node of a pair become consumers of the first. An edge links a method of an interface of the
// SDK to each method of an SDK type that implements it: a consumer of the interface method consumes every
// implementation.
type usage struct {
	items []item
	uses  []use
	edges [][2]string
}

func getGoKey(pkgPath, name string) string {
	return "go:" + pkgPath + "|" + name
}

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

func collectGoUsage(pkgs []*packages.Package, l layout) usage {
	var found usage
	seenItems := map[string]bool{}
	seenUses := map[use]bool{}
	seenPairs := map[string]bool{}
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
		}
	}
	return found
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
			if target := l.getGoTargetKey(pkg.TypesInfo.Uses[ident]); target != "" {
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

// tsOutput is what uses.mjs prints.
type tsOutput struct {
	Items    []item      `json:"items"`
	Uses     [][3]string `json:"uses"`
	Packages int         `json:"packages"`
	Files    int         `json:"files"`
}

func collectTSUsage(t *testing.T, l layout) (usage, tsOutput) {
	t.Helper()
	script := filepath.Join(getRealDir(t, load.RepoRootDir), "tools", "checks", "structure", "go", "single-consumer", "uses.mjs")
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

// place is what a use in a file counts as. A consumer is a plugin by its folder, an application by its
// folder, and any other package by its directory; a test, a story, a test kit, a fixture or a template is
// no consumer.
type place int

const (
	placeNone place = iota
	placeLibrary
	placeConsumer
)

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

// finding is an item used by nothing outside tests, or only by one plugin or the plugins of one family. A
// family is the part of a plugin folder before its first dash: agents, engines, views, readers, tools.
type finding struct {
	name     string
	position string
	verdict  string
}

func (f finding) String() string {
	return f.position + ": " + f.name + ": " + f.verdict
}

func getFamily(plugin string) string {
	family, _, _ := strings.Cut(plugin, "-")
	return family
}

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

func findSingleConsumers(parts []usage, l layout) []finding {
	consumers := getConsumers(parts, l)
	var found []finding
	seen := map[string]bool{}
	for _, part := range parts {
		for _, one := range part.items {
			if seen[one.Key] || (!strings.HasPrefix(one.Key, "rpc:") && isTestPath(getItemFile(one))) {
				continue
			}
			seen[one.Key] = true
			if verdict := getVerdict(consumers[one.Key]); verdict != "" {
				found = append(found, finding{name: one.Name, position: one.Position, verdict: verdict})
			}
		}
	}
	slices.SortFunc(found, func(a, b finding) int { return strings.Compare(a.name, b.name) })
	return found
}

func getVerdict(consumers map[string]bool) string {
	if len(consumers) == 0 {
		return "used by nothing outside tests: dead"
	}
	var plugins []string
	families := map[string]bool{}
	for consumer := range consumers {
		plugin, ok := strings.CutPrefix(consumer, "plugin ")
		if !ok {
			return ""
		}
		plugins = append(plugins, plugin)
		families[getFamily(plugin)] = true
	}
	slices.Sort(plugins)
	if len(plugins) == 1 {
		return "used only by plugin " + plugins[0] + ": belongs to that plugin"
	}
	if len(families) == 1 {
		return "used only by the " + getFamily(plugins[0]) + "-* plugins (" + strings.Join(plugins, ", ") + "): belongs to that family"
	}
	return ""
}

// allowance is an entry of allowlist.json: an item by its name and the reason it is kept in its package. An
// entry needs a reason of three words or more and must name a current finding.
type allowance struct {
	Item   string `json:"item"`
	Reason string `json:"reason"`
}

func readAllowList(t *testing.T) []allowance {
	t.Helper()
	file := filepath.Join(load.RepoRootDir, "tools", "checks", "structure", "go", "single-consumer", "allowlist.json")
	text, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var entries []allowance
	if err := json.Unmarshal(text, &entries); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	return entries
}

func applyAllowList(found []finding, entries []allowance) ([]finding, []string) {
	allowed := map[string]bool{}
	var problems []string
	for _, one := range entries {
		if len(strings.Fields(one.Reason)) < 3 {
			problems = append(problems, fmt.Sprintf("allowlist.json: %q gives no reason of three words or more", one.Item))
			continue
		}
		allowed[one.Item] = true
	}
	var kept []finding
	named := map[string]bool{}
	for _, one := range found {
		if allowed[one.name] {
			named[one.name] = true
			continue
		}
		kept = append(kept, one)
	}
	for _, one := range entries {
		if allowed[one.Item] && !named[one.Item] {
			problems = append(problems, fmt.Sprintf("allowlist.json: %q is not a finding; remove the entry", one.Item))
		}
	}
	return kept, problems
}

func checkSingleConsumers(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	l := newRepositoryLayout(t)
	var pkgs []*packages.Package
	for _, dir := range load.ListWorkspaceModules(t) {
		pkgs = append(pkgs, load.ListProductionPackages(workspace.ListModule(t, dir))...)
	}
	goUsage := collectGoUsage(pkgs, l)
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
	found, problems := applyAllowList(findSingleConsumers([]usage{goUsage, tsUsage}, l), readAllowList(t))
	for _, one := range problems {
		t.Error(one)
	}
	for _, one := range found {
		t.Error(one)
	}
}
