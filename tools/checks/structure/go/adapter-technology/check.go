package adaptertechnology

import (
	"slices"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getTechnologyFreeAdapters() map[string]string {
	return map[string]string{
		"internal/adapter/clock": "the machine clock itself, read through the time package",
	}
}

func isUseCaseImport(root, path string) bool {
	return load.IsWithin(path, root+"/usecase")
}

func getAdapterTechnologyFindings(pkgs []*packages.Package, root string, free map[string]string) (found []string, driven int) {
	for _, pkg := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, root), "/")
		isInternalAdapter := load.IsWithin(rel, "internal/adapter")
		if !isInternalAdapter && !load.IsWithin(rel, "adapter") {
			continue
		}
		load.IsDriving := false
		isTechnical := false
		for path := range pkg.Imports {
			isDriving = isDriving || isUseCaseImport(root, path)
			isTechnical = isTechnical || load.IsTechnologyImport(path)
		}
		switch {
		case isDriving && isInternalAdapter:
			found = append(found, rel+" calls a use case: a driving adapter lives under adapter/")
		case isDriving:
		default:
			driven++
			if _, isFree := free[rel]; !isTechnical && !isFree {
				found = append(found, rel+" is a driven adapter that imports no technology: it holds callbacks, not an integration")
			}
		}
	}
	slices.Sort(found)
	return found, driven
}

func checkDrivenAdaptersImportATechnology(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	pkgs := load.GetModulePackages(t, workspace, "libs/core/kernel")
	free := getTechnologyFreeAdapters()
	found, driven := getAdapterTechnologyFindings(pkgs, load.CoreImportPath, free)
	for _, one := range found {
		t.Error(one)
	}
	for name, reason := range free {
		if !slices.ContainsFunc(pkgs, func(pkg *packages.Package) bool { return pkg.PkgPath == load.CoreImportPath+"/"+name }) || reason == "" {
			t.Errorf("the entry %s names no package of the kernel or has no reason", name)
		}
	}
	if driven < 20 {
		t.Fatalf("%d driven adapters read: the rule is checked against nothing", driven)
	}
}
