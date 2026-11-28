package deadcode

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// scope is what the check reads: the Go packages the patterns match, relative to root, and the test files
// and Makefiles under root. env is added to the environment of every load. The analysis is the rapid
// type analysis that golang.org/x/tools/cmd/deadcode runs, from these roots: the init and main functions
// of every main package, the exported functions and methods of a package that a gomobile bind command of
// a Makefile names, the functions that a test file selects from a test-support package, and the methods
// of every type of a test-support package that a test file imports or that such a package imports. A
// test-support package is one whose directory name ends in "test" or is testsupport, or one under
// internal/testsupport. The body of a generic root is walked, and every function it names is a root
// too. Functions of generated files are not reported, nor an empty unexported method with no parameters
// and no results whose receiver implements an interface of its package: it marks the type and is never
// called.
type scope struct {
	root     string
	patterns []string
	env      []string
	external []externalSymbol
}

// configuration is one platform the packages are read for. A file that a build constraint leaves out of
// one platform is read by another, so a function is dead only when no configuration reaches it. The
// packages are read without cgo, so linux and darwin take the purego files of the desktop runtime. The
// darwin configuration reads the numen_debug files; linux and windows read the files without
// numen_debug. bound limits a configuration to the packages that gomobile binds.
type configuration struct {
	goos  string
	tags  []string
	bound bool
}

func getConfigurations() []configuration {
	return []configuration{
		{goos: "linux", tags: []string{"purego"}},
		{goos: "darwin", tags: []string{"purego", "numen_debug"}},
		{goos: "windows"},
		{goos: "android", bound: true},
	}
}

// gomobileBind matches a gomobile bind command of a Makefile.
var gomobileBind = regexp.MustCompile(`(?i)gomobile\)?\s+bind\s`)

func newRepositoryScope(t *testing.T) scope {
	t.Helper()
	root := getRealDir(t, load.RepoRootDir)
	var patterns []string
	for _, dir := range load.ListWorkspaceModules(t) {
		resolved := getRealDir(t, dir)
		rel, err := filepath.Rel(root, resolved)
		if err != nil {
			t.Fatal(err)
		}
		rel = filepath.ToSlash(rel)
		first, _, _ := strings.Cut(rel, "/")
		if slices.Contains([]string{"libs", "plugins", "apps"}, first) {
			patterns = append(patterns, "./"+rel+"/...")
		}
	}
	if len(patterns) == 0 {
		t.Fatal("go.work uses no module under libs, plugins or apps")
	}
	return scope{root: root, patterns: patterns, external: readExternalSymbols(t, filepath.Join(load.RepoRootDir, "tools/checks/structure/go/deadcode/used-by-other-repositories.json"))}
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

// testUse is a name a test file selects from an imported package: `servertest.NewHandler`.
type testUse struct {
	pkgPath string
	name    string
}

// testFile is the imports of one test file and the selectors on a package name it holds.
type testFile struct {
	imports   map[string]string
	selectors map[[2]string]bool
}

func findUnreachableFunctions(t *testing.T, s scope) (dead []load.Finding, missing []string) {
	t.Helper()
	tests := readTestFiles(t, s.root)
	bound := findBoundPackages(t, s.root)
	declared := map[string]load.Finding{}
	reached := map[string]bool{}
	for _, one := range getConfigurations() {
		patterns := s.patterns
		if one.bound {
			patterns = bound
		}
		if len(patterns) == 0 {
			continue
		}
		readConfiguration(t, s, one, patterns, tests, declared, reached)
	}
	var found []load.Finding
	for key, finding := range declared {
		if !reached[key] && !isException(finding.Name) && !isExternal(s.external, finding.Name) {
			found = append(found, finding)
		}
	}
	return load.SortFindings(found), findMissingExternalSymbols(declared, s.external)
}

func listExceptions() []string {
	// numen:kept-list #2126 groundwork of the online plugin store (CachedSource, snapshotcache, packagecache), kept on the user's decision of 2026-11-15 until the store lands
	return []string{
		"github.com/jiva-studio/numen/libs/plugin/catalog/catalog.NewCachedSource",
		"github.com/jiva-studio/numen/libs/plugin/catalog/catalog.(*CachedSource).",
		"github.com/jiva-studio/numen/libs/plugin/catalog/snapshotcache.",
		"github.com/jiva-studio/numen/libs/plugin/catalog/packagecache.",
	}
}

func isException(name string) bool {
	return slices.ContainsFunc(listExceptions(), func(entry string) bool {
		if strings.HasSuffix(entry, ".") {
			return strings.HasPrefix(name, entry)
		}
		return name == entry
	})
}

func readConfiguration(t *testing.T, s scope, one configuration, patterns []string, tests []testFile, declared map[string]load.Finding, reached map[string]bool) {
	t.Helper()
	env := append(os.Environ(), "GOOS="+one.goos, "CGO_ENABLED=0")
	cfg := &packages.Config{
		Dir:        s.root,
		Env:        append(env, s.env...),
		BuildFlags: []string{"-tags=" + strings.Join(one.tags, ",")},
		Mode:       packages.LoadAllSyntax | packages.NeedModule,
	}
	loaded, err := packages.Load(cfg, patterns...)
	if err != nil {
		t.Fatalf("%s: %v", one.goos, err)
	}
	var initial []*packages.Package
	for _, pkg := range loaded {
		if !hasNoFiles(pkg) {
			initial = append(initial, pkg)
		}
	}
	if len(initial) == 0 {
		t.Fatalf("%s: the patterns %v match no package", one.goos, patterns)
	}
	var failures []string
	packages.Visit(initial, nil, func(pkg *packages.Package) {
		for _, e := range pkg.Errors {
			failures = append(failures, e.Error())
		}
	})
	if len(failures) > 0 {
		t.Fatalf("%s: the packages do not type-check:\n%s", one.goos, strings.Join(failures[:min(len(failures), 20)], "\n"))
	}
	prog, ssaPackages := ssautil.AllPackages(initial, ssa.InstantiateGenerics)
	prog.Build()
	roots, live := collectRoots(prog, initial, ssaPackages, one.bound, tests)
	for _, pos := range live {
		reached[getKey(s.root, prog.Fset.Position(pos))] = true
	}
	if len(roots) > 0 {
		for fn := range rta.Analyze(roots, false).Reachable {
			if fn.Pos().IsValid() {
				reached[getKey(s.root, prog.Fset.Position(fn.Pos()))] = true
			}
		}
	}
	for _, pkg := range initial {
		collectDeclaredFunctions(s.root, pkg, declared)
	}
}

func hasNoFiles(pkg *packages.Package) bool {
	if len(pkg.CompiledGoFiles) > 0 {
		return false
	}
	return !slices.ContainsFunc(pkg.Errors, func(e packages.Error) bool {
		return !strings.Contains(e.Msg, "build constraints exclude all Go files") && !strings.Contains(e.Msg, "no non-test Go files")
	})
}

// roots is where the analysis starts. The rapid type analysis reaches the instances of a generic
// function, not its body, so a generic root is kept apart and its body is walked by expandGenerics.
type roots struct {
	functions []*ssa.Function
	generics  []*ssa.Function
}

func (r *roots) add(fn *ssa.Function) {
	switch {
	case fn == nil:
	case isGeneric(fn):
		r.generics = append(r.generics, fn)
	default:
		r.functions = append(r.functions, fn)
	}
}

func isGeneric(fn *ssa.Function) bool {
	if fn.TypeParams().Len() > 0 && len(fn.TypeArgs()) == 0 {
		return true
	}
	if recv := fn.Signature.Recv(); recv != nil && fn.Origin() == nil && hasTypeParams(recv.Type()) {
		return true
	}
	return fn.Parent() != nil && isGeneric(fn.Parent())
}

func hasTypeParams(t types.Type) bool {
	if pointer, ok := t.(*types.Pointer); ok {
		t = pointer.Elem()
	}
	named, ok := t.(*types.Named)
	return ok && named.TypeParams().Len() > 0
}

func collectRoots(prog *ssa.Program, initial []*packages.Package, ssaPackages []*ssa.Package, bound bool, tests []testFile) ([]*ssa.Function, []token.Pos) {
	var r roots
	for _, pkg := range ssautil.MainPackages(slices.DeleteFunc(slices.Clone(ssaPackages), func(p *ssa.Package) bool { return p == nil })) {
		r.add(pkg.Func("init"))
		r.add(pkg.Func("main"))
	}
	byPath := map[string]*ssa.Package{}
	names := map[string]string{}
	imports := map[string][]string{}
	for i, pkg := range initial {
		if ssaPackages[i] == nil {
			continue
		}
		byPath[pkg.PkgPath] = ssaPackages[i]
		names[pkg.PkgPath] = pkg.Name
		imports[pkg.PkgPath] = slices.Collect(maps.Keys(pkg.Imports))
		if bound && pkg.Name != "main" {
			addBoundPackage(prog, &r, ssaPackages[i])
		}
	}
	for _, pkgPath := range listTestSupportImports(tests, names, imports) {
		addTestSupportPackage(prog, &r, byPath[pkgPath])
	}
	for _, use := range listTestUses(tests, names) {
		if pkg, ok := byPath[use.pkgPath]; ok {
			if member, ok := pkg.Members[use.name]; ok {
				addMember(prog, &r, member, false)
			}
		}
	}
	live := expandGenerics(&r)
	return r.functions, live
}

func addBoundPackage(prog *ssa.Program, r *roots, pkg *ssa.Package) {
	r.add(pkg.Func("init"))
	for _, member := range pkg.Members {
		if member.Object() != nil && member.Object().Exported() {
			addMember(prog, r, member, true)
		}
	}
}

func addTestSupportPackage(prog *ssa.Program, r *roots, pkg *ssa.Package) {
	if pkg == nil {
		return
	}
	r.add(pkg.Func("init"))
	for _, member := range pkg.Members {
		if typ, ok := member.(*ssa.Type); ok {
			addMethods(prog, r, typ, false)
		}
	}
}

func addMember(prog *ssa.Program, r *roots, member ssa.Member, exportedOnly bool) {
	switch m := member.(type) {
	case *ssa.Function:
		r.add(m)
	case *ssa.Type:
		addMethods(prog, r, m, exportedOnly)
	}
}

func addMethods(prog *ssa.Program, r *roots, typ *ssa.Type, exportedOnly bool) {
	named, ok := typ.Type().(*types.Named)
	if !ok || types.IsInterface(named) {
		return
	}
	for method := range named.Methods() {
		if exportedOnly && !method.Exported() {
			continue
		}
		if named.TypeParams().Len() > 0 {
			r.add(prog.FuncValue(method))
			continue
		}
		recv := types.Type(named)
		if _, pointer := method.Signature().Recv().Type().(*types.Pointer); pointer {
			recv = types.NewPointer(named)
		}
		if sel := prog.MethodSets.MethodSet(recv).Lookup(method.Pkg(), method.Name()); sel != nil {
			r.add(prog.MethodValue(sel))
		}
	}
}

func expandGenerics(r *roots) []token.Pos {
	var live []token.Pos
	walked := map[*ssa.Function]bool{}
	queue := slices.Clone(r.generics)
	for len(queue) > 0 {
		fn := queue[0]
		queue = queue[1:]
		if walked[fn] {
			continue
		}
		walked[fn] = true
		live = append(live, fn.Pos())
		queue = append(queue, fn.AnonFuncs...)
		for _, named := range listNamedFunctions(fn) {
			if isGeneric(named) {
				queue = append(queue, named)
			} else {
				r.functions = append(r.functions, named)
			}
		}
	}
	return live
}

func listNamedFunctions(fn *ssa.Function) []*ssa.Function {
	var named []*ssa.Function
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			for _, operand := range instr.Operands(nil) {
				callee, ok := (*operand).(*ssa.Function)
				if !ok {
					continue
				}
				if origin := callee.Origin(); origin != nil {
					callee = origin
				}
				named = append(named, callee)
			}
		}
	}
	return named
}

func listTestSupportImports(tests []testFile, names map[string]string, imports map[string][]string) []string {
	var queue []string
	for _, file := range tests {
		for pkgPath := range file.imports {
			queue = append(queue, pkgPath)
		}
	}
	var imported []string
	visited := map[string]bool{}
	for len(queue) > 0 {
		pkgPath := queue[0]
		queue = queue[1:]
		if _, ok := names[pkgPath]; !ok || visited[pkgPath] || !isTestSupport(pkgPath) {
			continue
		}
		visited[pkgPath] = true
		imported = append(imported, pkgPath)
		queue = append(queue, imports[pkgPath]...)
	}
	return imported
}

func listTestUses(tests []testFile, names map[string]string) []testUse {
	var uses []testUse
	for _, file := range tests {
		for pkgPath, local := range file.imports {
			name, ok := names[pkgPath]
			if !ok || !isTestSupport(pkgPath) {
				continue
			}
			if local == "" {
				local = name
			}
			for pair := range file.selectors {
				if pair[0] == local {
					uses = append(uses, testUse{pkgPath: pkgPath, name: pair[1]})
				}
			}
		}
	}
	return uses
}

func readTestFiles(t *testing.T, root string) []testFile {
	t.Helper()
	var files []testFile
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			base := entry.Name()
			if name != root && (strings.HasPrefix(base, ".") || base == "node_modules" || base == "testdata" || base == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, "_test.go") {
			return nil
		}
		parsed, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		files = append(files, readTestFile(parsed))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func readTestFile(file *ast.File) testFile {
	read := testFile{imports: map[string]string{}, selectors: map[[2]string]bool{}}
	for _, spec := range file.Imports {
		local := ""
		if spec.Name != nil {
			local = spec.Name.Name
		}
		read.imports[strings.Trim(spec.Path.Value, `"`)] = local
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if sel, ok := node.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				read.selectors[[2]string{ident.Name, sel.Sel.Name}] = true
			}
		}
		return true
	})
	return read
}

func findBoundPackages(t *testing.T, root string) []string {
	t.Helper()
	dir, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dir.Close(); err != nil {
			t.Error(err)
		}
	}()
	tree := dir.FS()
	var bound []string
	err = fs.WalkDir(tree, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			base := entry.Name()
			if name != "." && (strings.HasPrefix(base, ".") || base == "node_modules" || base == "testdata") {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Name() != "Makefile" {
			return nil
		}
		text, err := fs.ReadFile(tree, name)
		if err != nil {
			return err
		}
		for line := range strings.SplitSeq(strings.ReplaceAll(string(text), "\\\n", " "), "\n") {
			if !gomobileBind.MatchString(line) {
				continue
			}
			fields := strings.Fields(line)
			last := fields[len(fields)-1]
			if !strings.HasPrefix(last, "./") {
				continue
			}
			bound = append(bound, "./"+path.Join(path.Dir(name), last))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return bound
}

func collectDeclaredFunctions(root string, pkg *packages.Package, declared map[string]load.Finding) {
	var interfaces []*types.Interface
	for _, name := range pkg.Types.Scope().Names() {
		typeName, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		if iface, ok := typeName.Type().Underlying().(*types.Interface); ok {
			interfaces = append(interfaces, iface)
		}
	}
	for _, file := range pkg.Syntax {
		if ast.IsGenerated(file) {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name == "_" {
				continue
			}
			obj, ok := pkg.TypesInfo.Defs[fn.Name].(*types.Func)
			if !ok || isMarkerMethod(fn, obj, interfaces) {
				continue
			}
			position := pkg.Fset.Position(fn.Name.Pos())
			key := getKey(root, position)
			declared[key] = load.Finding{Position: getRelative(root, position), Name: getFunctionName(pkg.PkgPath, obj)}
		}
	}
}

func isMarkerMethod(decl *ast.FuncDecl, obj *types.Func, interfaces []*types.Interface) bool {
	sig := obj.Signature()
	if sig.Recv() == nil || obj.Exported() || sig.Params().Len() > 0 || sig.Results().Len() > 0 {
		return false
	}
	if decl.Body == nil || len(decl.Body.List) > 0 {
		return false
	}
	return slices.ContainsFunc(interfaces, func(iface *types.Interface) bool {
		return types.Implements(sig.Recv().Type(), iface)
	})
}

func getFunctionName(pkgPath string, obj *types.Func) string {
	recv := obj.Signature().Recv()
	if recv == nil {
		return pkgPath + "." + obj.Name()
	}
	qualifier := func(*types.Package) string { return "" }
	return fmt.Sprintf("%s.(%s).%s", pkgPath, types.TypeString(recv.Type(), qualifier), obj.Name())
}

func getKey(root string, position token.Position) string {
	return fmt.Sprintf("%s:%d", getRelative(root, position), position.Column)
}

func getRelative(root string, position token.Position) string {
	rel, err := filepath.Rel(root, position.Filename)
	if err != nil {
		rel = position.Filename
	}
	return fmt.Sprintf("%s:%d", filepath.ToSlash(rel), position.Line)
}

func isTestSupport(pkgPath string) bool {
	base := path.Base(pkgPath)
	return strings.HasSuffix(base, "test") || base == "testsupport" || strings.Contains(pkgPath+"/", "/internal/testsupport/")
}

// externalSymbol is a function that another repository calls, listed in used-by-other-repositories.json.
type externalSymbol struct {
	Package string `json:"package"`
	Name    string `json:"name"`
}

func (e externalSymbol) getFullName() string { return e.Package + "." + e.Name }

func readExternalSymbols(t *testing.T, file string) []externalSymbol {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Symbols []externalSymbol `json:"symbols"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	return manifest.Symbols
}

func findMissingExternalSymbols(declared map[string]load.Finding, external []externalSymbol) []string {
	names := map[string]bool{}
	for _, finding := range declared {
		names[finding.Name] = true
	}
	var missing []string
	for _, one := range external {
		if !names[one.getFullName()] {
			missing = append(missing, one.getFullName())
		}
	}
	return missing
}

func isExternal(external []externalSymbol, name string) bool {
	return slices.ContainsFunc(external, func(one externalSymbol) bool { return one.getFullName() == name })
}
