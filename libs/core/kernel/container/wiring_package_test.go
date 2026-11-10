package container

import (
	"regexp"
	"slices"
	"testing"

	"golang.org/x/tools/go/packages"
)

// compositionRoot matches the packages of an application that compose it: apps/<name>/cmd and
// apps/<name>/app, and what lies below them.
var compositionRoot = regexp.MustCompile(`^` + regexp.QuoteMeta(numenRoot) + `apps/[^/]+/(cmd|app)(/|$)`)

func getWiringPackages(pkgs []*packages.Package, root string, isExempt func(path string) bool) ([]finding, int) {
	ports := listPortInterfaces(pkgs, root)
	driven := map[string]bool{}
	for _, pkg := range pkgs {
		if getImplementation(pkg, ports).name != "" {
			driven[pkg.PkgPath] = true
		}
	}
	var found []finding
	exempted := 0
	for _, pkg := range pkgs {
		if len(pkg.Syntax) == 0 {
			continue
		}
		hasUseCase, hasDriven := false, false
		for path := range pkg.Imports {
			hasUseCase = hasUseCase || isInLayer(path, root, "usecase")
			hasDriven = hasDriven || driven[path]
		}
		if !hasUseCase || !hasDriven {
			continue
		}
		if isExempt(pkg.PkgPath) {
			exempted++
			continue
		}
		found = append(found, newFinding(pkg.Fset, pkg.Syntax[0].Package, pkg.PkgPath))
	}
	return sortFindings(found), exempted
}

func isCompositionRoot(path string) bool {
	return isWithin(path, coreImportPath+"/container") || compositionRoot.MatchString(path)
}

func checkNoWiringPackageExistsOutsideTheContainer(t *testing.T, workspace workspacePackages) {
	t.Helper()
	var pkgs []*packages.Package
	for _, rel := range []string{"libs/core/kernel", "apps/desktop", "apps/mobile"} {
		pkgs = append(pkgs, getModulePackages(t, workspace, rel)...)
	}
	found, exempted := getWiringPackages(pkgs, coreImportPath, isCompositionRoot)
	for _, one := range found {
		t.Errorf("%s: %s builds use cases from driven adapters; the wiring is the one package container", one.position, one.name)
	}

	// A walk that finds no composition root with the shape checks nothing.
	if exempted == 0 {
		t.Fatal("no composition root has the shape of wiring: the rule is checked against nothing")
	}
}

// The rule rejects a package that imports a use case and a driven adapter, and accepts a driving adapter,
// a driven adapter, a package that imports one of the two, and the exempt composition root.
func TestWiringPackageRule_RejectsAPackageThatBuildsUseCasesFromDrivenAdapters(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"port/noteport/noteport.go": "package noteport\n\ntype Notes interface{ Read() string }\n",
		"usecase/find/find.go": `package find

import "fixture/port/noteport"

type Find struct{ notes noteport.Notes }

func New(notes noteport.Notes) *Find { return &Find{notes: notes} }
`,
		"adapter/store/store.go": "package store\n\ntype Store struct{}\n\nfunc (*Store) Read() string { return \"\" }\n",
		"container/container.go": `package container

import (
	"fixture/adapter/store"
	"fixture/usecase/find"
)

func New() *find.Find { return find.New(&store.Store{}) }
`,
		"internal/hub/hub.go": `package hub

import (
	"fixture/adapter/store"
	"fixture/usecase/find"
)

func Build() *find.Find { return find.New(&store.Store{}) }
`,
		"adapter/driving/driving.go": `package driving

import "fixture/usecase/find"

func Run(f *find.Find) {}
`,
		"internal/stores/stores.go": `package stores

import "fixture/adapter/store"

func Open() *store.Store { return &store.Store{} }
`,
	})
	found, exempted := getWiringPackages(loadFixture(t, dir), "fixture", func(path string) bool { return path == "fixture/container" })
	want := []string{"hub.go:1 fixture/internal/hub"}
	if got := formatFindings(found); !slices.Equal(got, want) || exempted != 1 {
		t.Errorf("the rule rejects %v with %d exempt, want %v with 1", got, exempted, want)
	}
}

func TestCompositionRootRule_ExemptsTheContainerAndTheApplicationRoots(t *testing.T) {
	for _, tc := range []struct {
		path     string
		isExempt bool
	}{
		{coreImportPath + "/container", true},
		{coreImportPath + "/container/vault", true},
		{numenRoot + "apps/desktop/cmd/numen", true},
		{numenRoot + "apps/desktop/app", true},
		{numenRoot + "apps/mobile/app/sub", true},
		{numenRoot + "apps/desktop/internal/platform", false},
		{coreImportPath + "/internal/adapterlib/wire", false},
		{coreImportPath + "/adapter/window/editor", false},
		{numenRoot + "apps/desktop/application", false},
	} {
		if got := isCompositionRoot(tc.path); got != tc.isExempt {
			t.Errorf("%s is exempt: %v, want %v", tc.path, got, tc.isExempt)
		}
	}
}
