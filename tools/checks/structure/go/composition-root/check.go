package compositionroot

import (
	"go/ast"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

func getRootSubpackages(pkgs []*packages.Package, rootPath string) ([]load.Finding, bool) {
	var found []load.Finding
	isRootRead := false
	for _, pkg := range pkgs {
		if pkg.PkgPath == rootPath {
			isRootRead = true
		}
		if !strings.HasPrefix(pkg.PkgPath, rootPath+"/") || len(pkg.Syntax) == 0 {
			continue
		}
		files := slices.Clone(pkg.Syntax)
		slices.SortFunc(files, func(a, b *ast.File) int {
			return strings.Compare(pkg.Fset.Position(a.Package).Filename, pkg.Fset.Position(b.Package).Filename)
		})
		found = append(found, load.NewFinding(pkg.Fset, files[0].Package, pkg.PkgPath))
	}
	return load.SortFindings(found), isRootRead
}

func checkCompositionRootIsOnePackage(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	found, isRootRead := getRootSubpackages(load.GetModulePackages(t, workspace, "libs/core/kernel"), load.CoreImportPath+"/container")
	for _, one := range found {
		t.Errorf("%s: %s is a package below the composition root; the root is the one package container", one.Position, one.Name)
	}
	if !isRootRead {
		t.Fatal("the package container was not read: the rule is checked against nothing")
	}
}
