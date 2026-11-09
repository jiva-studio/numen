package container

import (
	"go/ast"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// getRootSubpackages returns each package below the composition root, at its first file, and whether the
// root itself was among the packages.
func getRootSubpackages(pkgs []*packages.Package, rootPath string) ([]finding, bool) {
	var found []finding
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
		found = append(found, newFinding(pkg.Fset, files[0].Package, pkg.PkgPath))
	}
	return sortFindings(found), isRootRead
}

func checkCompositionRootIsOnePackage(t *testing.T, workspace workspacePackages) {
	t.Helper()
	found, isRootRead := getRootSubpackages(getModulePackages(t, workspace, "libs/core/kernel"), coreImportPath+"/container")
	for _, one := range found {
		t.Errorf("%s: %s is a package below the composition root; the root is the one package container", one.position, one.name)
	}
	if !isRootRead {
		t.Fatal("the package container was not read: the rule is checked against nothing")
	}
}

// The rule rejects a package below the root, and accepts the root and a package beside it.
func TestCompositionRootRule_RejectsAPackageBelowTheRoot(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"container/container.go":         "package container\n\nfunc New() {}\n",
		"container/settings/settings.go": "package settings\n\nfunc Load() {}\n",
		"containerkit/kit.go":            "package containerkit\n\nfunc Build() {}\n",
	})
	found, isRootRead := getRootSubpackages(loadFixture(t, dir), "fixture/container")
	if want := []string{"settings.go:1 fixture/container/settings"}; !isRootRead || !slices.Equal(formatFindings(found), want) {
		t.Errorf("the rule rejects %v with the root read %v, want %v", formatFindings(found), isRootRead, want)
	}
}
