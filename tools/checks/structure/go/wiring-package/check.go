package wiringpackage

import (
	"regexp"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// compositionRoot matches the packages of an application that compose it: apps/<name>/cmd and
// apps/<name>/app, and what lies below them.
var compositionRoot = regexp.MustCompile(`^` + regexp.QuoteMeta(load.NumenRoot) + `apps/[^/]+/(cmd|app)(/|$)`)

func getWiringPackages(pkgs []*packages.Package, root string, isExempt func(path string) bool) ([]load.Finding, int) {
	ports := load.ListPortInterfaces(pkgs, root)
	driven := map[string]bool{}
	for _, pkg := range pkgs {
		if load.GetImplementation(pkg, ports).Name != "" {
			driven[pkg.PkgPath] = true
		}
	}
	var found []load.Finding
	exempted := 0
	for _, pkg := range pkgs {
		if len(pkg.Syntax) == 0 {
			continue
		}
		hasUseCase, hasDriven := false, false
		for path := range pkg.Imports {
			hasUseCase = hasUseCase || load.IsInLayer(path, root, "usecase")
			hasDriven = hasDriven || driven[path]
		}
		if !hasUseCase || !hasDriven {
			continue
		}
		if isExempt(pkg.PkgPath) {
			exempted++
			continue
		}
		found = append(found, load.NewFinding(pkg.Fset, pkg.Syntax[0].Package, pkg.PkgPath))
	}
	return load.SortFindings(found), exempted
}

func isCompositionRoot(path string) bool {
	return load.IsWithin(path, load.CoreImportPath+"/container") || compositionRoot.MatchString(path)
}

func checkNoWiringPackageExistsOutsideTheContainer(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	var pkgs []*packages.Package
	for _, rel := range []string{"libs/core/kernel", "apps/desktop", "apps/mobile"} {
		pkgs = append(pkgs, load.GetModulePackages(t, workspace, rel)...)
	}
	found, exempted := getWiringPackages(pkgs, load.CoreImportPath, isCompositionRoot)
	for _, one := range found {
		t.Errorf("%s: %s builds use cases from driven adapters; the wiring is the one package container", one.Position, one.Name)
	}

	// A walk that finds no composition root with the shape checks nothing.
	if exempted == 0 {
		t.Fatal("no composition root has the shape of wiring: the rule is checked against nothing")
	}
}
