package sdkprotocolencapsulation

import (
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// protocolLeak is a protocol type mentioned by the type of an exported identifier of a public package.
type protocolLeak struct {
	subject      string
	protocolType string
}

// leakWalk collects the protocol types that the public API of one set of packages mentions.
type leakWalk struct {
	protocolRoot string
	familyPrefix string
	subject      string
	visited      map[*types.TypeName]bool
	found        map[protocolLeak]bool
}

func isInternalPackage(path string) bool {
	return slices.Contains(strings.Split(path, "/"), "internal")
}

func (w *leakWalk) isProtocolPackage(path string) bool {
	return load.IsWithin(path, w.protocolRoot)
}

func (w *leakWalk) walkNamed(named *types.Named) {
	origin := named.Origin().Obj()
	if origin.Pkg() == nil {
		return
	}
	if w.isProtocolPackage(origin.Pkg().Path()) {
		w.found[protocolLeak{subject: w.subject, protocolType: origin.Pkg().Path() + "." + origin.Name()}] = true
		return
	}
	for t := range named.TypeArgs().Types() {
		w.walk(t)
	}
	if !strings.HasPrefix(origin.Pkg().Path(), w.familyPrefix) || w.visited[origin] {
		return
	}
	w.visited[origin] = true
	outer := w.subject
	w.subject = origin.Pkg().Path() + "." + origin.Name()
	defer func() { w.subject = outer }()
	w.walk(named.Origin().Underlying())
	methods := types.NewMethodSet(types.NewPointer(named.Origin()))
	for method := range methods.Methods() {
		if method := method.Obj(); method.Exported() {
			w.walk(method.Type())
		}
	}
}

func (w *leakWalk) walk(t types.Type) {
	switch current := types.Unalias(t).(type) {
	case *types.Named:
		w.walkNamed(current)
	case *types.Pointer:
		w.walk(current.Elem())
	case *types.Slice:
		w.walk(current.Elem())
	case *types.Array:
		w.walk(current.Elem())
	case *types.Chan:
		w.walk(current.Elem())
	case *types.Map:
		w.walk(current.Key())
		w.walk(current.Elem())
	case *types.Struct:
		for field := range current.Fields() {
			if field.Exported() {
				w.walk(field.Type())
			}
		}
	case *types.Tuple:
		for v := range current.Variables() {
			w.walk(v.Type())
		}
	case *types.Signature:
		w.walk(current.Params())
		w.walk(current.Results())
		for tparam := range current.TypeParams().TypeParams() {
			w.walk(tparam.Constraint())
		}
	case *types.Interface:
		for method := range current.ExplicitMethods() {
			if method.Exported() {
				w.walk(method.Type())
			}
		}
		for etyp := range current.EmbeddedTypes() {
			w.walk(etyp)
		}
	case *types.Union:
		for term := range current.Terms() {
			w.walk(term.Type())
		}
	case *types.TypeParam:
		w.walk(current.Constraint())
	}
}

func getProtocolLeaks(pkgs []*packages.Package, protocolRoot, familyPrefix string) ([]protocolLeak, int) {
	walk := &leakWalk{
		protocolRoot: protocolRoot,
		familyPrefix: familyPrefix,
		visited:      map[*types.TypeName]bool{},
		found:        map[protocolLeak]bool{},
	}
	examined := 0
	for _, pkg := range pkgs {
		if pkg.Types == nil || isInternalPackage(pkg.PkgPath) || walk.isProtocolPackage(pkg.PkgPath) {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			object := scope.Lookup(name)
			if !object.Exported() {
				continue
			}
			examined++
			walk.subject = pkg.PkgPath + "." + name
			walk.walk(object.Type())
		}
	}
	leaks := make([]protocolLeak, 0, len(walk.found))
	for one := range walk.found {
		leaks = append(leaks, one)
	}
	slices.SortFunc(leaks, func(a, b protocolLeak) int {
		if byName := strings.Compare(a.subject, b.subject); byName != 0 {
			return byName
		}
		return strings.Compare(a.protocolType, b.protocolType)
	})
	return leaks, examined
}

func loadAPIPackages(t *testing.T, dir string, env []string) []*packages.Package {
	t.Helper()
	config := &packages.Config{
		Dir:  dir,
		Env:  append(os.Environ(), env...),
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	}
	loaded, err := packages.Load(config, "./...")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range loaded {
		for _, one := range pkg.Errors {
			t.Errorf("%s does not load: %v", pkg.ID, one)
		}
	}
	return loaded
}

func checkSDKNamesNoProtocolType(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	var pkgs []*packages.Package
	modules := 0
	root, err := filepath.Abs(filepath.Join(load.RepoRootDir, "libs", "plugin", "sdk-go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range load.ListWorkspaceModules(t) {
		relative, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(relative, "..") {
			continue
		}
		modules++
		for _, pkg := range workspace.ListModule(t, dir) {
			if load.IsTestVariant(pkg) {
				continue
			}
			for _, one := range pkg.Errors {
				t.Errorf("%s does not load: %v", pkg.ID, one)
			}
			pkgs = append(pkgs, pkg)
		}
	}
	leaks, examined := getProtocolLeaks(pkgs, load.Wire, load.SdkModule)
	for _, one := range leaks {
		t.Errorf("%s mentions the protocol type %s", one.subject, one.protocolType)
	}

	// A load that finds few modules, packages or identifiers, or an SDK that never imports the
	// protocol, checks nothing.
	importsProtocol := 0
	for _, pkg := range pkgs {
		for path := range pkg.Imports {
			if load.IsWithin(path, load.Wire) {
				importsProtocol++
				break
			}
		}
	}
	if modules == 0 || len(pkgs) == 0 || examined == 0 || importsProtocol == 0 {
		t.Fatalf("%d modules, %d packages, %d exported identifiers and %d packages importing the protocol: the rule is checked against nothing", modules, len(pkgs), examined, importsProtocol)
	}
}
