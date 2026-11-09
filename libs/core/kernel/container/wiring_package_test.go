package container

import (
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// listWiringWords returns the names a package takes when it builds the objects of the program and hands them
// to each other.
func listWiringWords() []string {
	return []string{"wire", "wiring", "compose", "composition", "assembly", "assemble", "bootstrap", "inject", "di", "container"}
}

// getWiringPackages returns each package, at its first file, whose last path element is a wiring word and
// whose path is not one of the allowed roots, and whether an allowed root was among the packages.
func getWiringPackages(pkgs []*packages.Package, roots []string) ([]finding, bool) {
	var found []finding
	isRootRead := false
	for _, pkg := range pkgs {
		if slices.Contains(roots, pkg.PkgPath) {
			isRootRead = true
			continue
		}
		if len(pkg.Syntax) == 0 || !slices.Contains(listWiringWords(), strings.ToLower(path.Base(pkg.PkgPath))) {
			continue
		}
		found = append(found, newFinding(pkg.Fset, pkg.Syntax[0].Package, pkg.PkgPath))
	}
	return sortFindings(found), isRootRead
}

func checkNoWiringPackageExistsOutsideTheContainer(t *testing.T, workspace workspacePackages) {
	t.Helper()
	root, err := filepath.Abs(repoRootDir)
	if err != nil {
		t.Fatal(err)
	}
	var pkgs []*packages.Package
	for _, dir := range listWorkspaceModules(t) {
		// A plugin names its own packages after its own domain: the container of an EPUB file is not wiring.
		if rel, err := filepath.Rel(root, dir); err != nil || isWithin(filepath.ToSlash(rel), "plugins") {
			continue
		}
		pkgs = append(pkgs, listProductionPackages(workspace.listModule(t, dir))...)
	}
	found, isRootRead := getWiringPackages(pkgs, []string{coreImportPath + "/container"})
	for _, one := range found {
		t.Errorf("%s: %s is a wiring package; the wiring is the one package container", one.position, one.name)
	}
	if !isRootRead {
		t.Fatal("the package container was not read: the rule is checked against nothing")
	}
}

// The rule rejects a package named for wiring anywhere but the root, and accepts the root and a package
// of another name.
func TestWiringPackageRule_RejectsAWiringPackageOutsideTheContainer(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"container/container.go":       "package container\n\nfunc New() {}\n",
		"internal/wiring/wiring.go":    "package wiring\n\nfunc Build() {}\n",
		"adapter/window/assembly/a.go": "package assembly\n\nfunc Build() {}\n",
		"adapter/window/window.go":     "package window\n\nfunc Serve() {}\n",
		"wirelength/wirelength.go":     "package wirelength\n\nfunc Measure() {}\n",
	})
	found, isRootRead := getWiringPackages(loadFixture(t, dir), []string{"fixture/container"})
	want := []string{"a.go:1 fixture/adapter/window/assembly", "wiring.go:1 fixture/internal/wiring"}
	if !isRootRead || !slices.Equal(formatFindings(found), want) {
		t.Errorf("the rule rejects %v with the root read %v, want %v", formatFindings(found), isRootRead, want)
	}
}
