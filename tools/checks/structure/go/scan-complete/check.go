package scancomplete

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func listLayerGroups() []string {
	return []string{"domain", "port", "usecase", "adapter", "internal/adapter", "internal/adapterlib", "internal/testsupport", "container"}
}

func getHelperPackages() map[string]string {
	return map[string]string{
		"internal/appearance": "builds the appearance section of a page from settings values",
		"internal/csp":        "builds the content security policy string of a window",
		"internal/testonly":   "the grant that only tests may hold",
		"internal/text":       "byte-range arithmetic over text",
		"internal/vector":     "quantises a vector to bytes and bits",
	}
}

func getLayerGroup(rel string, helpers map[string]string) (group string, ok bool) {
	var found []string
	for _, one := range listLayerGroups() {
		if load.IsWithin(rel, one) {
			found = append(found, one)
		}
	}
	if _, isHelper := helpers[rel]; isHelper {
		found = append(found, rel)
	}
	found = slices.Compact(found)
	if len(found) != 1 {
		return strings.Join(found, "+"), false
	}
	return found[0], true
}

func listProductionDirs(root string) ([]string, error) {
	withSource := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" || entry.Name() == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			rel, relErr := filepath.Rel(root, filepath.Dir(path))
			if relErr != nil {
				return relErr
			}
			withSource[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	dirs := make([]string, 0, len(withSource))
	for dir := range withSource {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	return dirs, err
}

func getUnplacedPackages(dirs []string, helpers map[string]string) []string {
	var found []string
	for _, dir := range dirs {
		if group, ok := getLayerGroup(dir, helpers); !ok {
			found = append(found, dir+" is in "+strconv.Quote(group)+" groups, want exactly one")
		}
	}
	return found
}

func getUseCaseImports(pkgs []*packages.Package) (imports map[string][]string, read int) {
	imports = map[string][]string{}
	for _, pkg := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, load.CoreImportPath), "/")
		if !load.IsWithin(rel, "usecase") {
			continue
		}
		read++
		for path := range pkg.Imports {
			if load.IsWithin(path, load.CoreImportPath) {
				target := strings.TrimPrefix(strings.TrimPrefix(path, load.CoreImportPath), "/")
				imports[rel] = append(imports[rel], target)
			}
		}
	}
	return imports, read
}

func getBadUseCaseImports(imports map[string][]string, helpers map[string]string) []string {
	var found []string
	for from, targets := range imports {
		for _, target := range targets {
			group, _ := getLayerGroup(target, helpers)
			if _, isHelper := helpers[target]; isHelper {
				continue
			}
			if group != "domain" && group != "port" && group != "usecase" {
				found = append(found, from+" imports "+target+" ("+group+"): a use case imports the domain, ports and its own helpers")
			}
		}
	}
	slices.Sort(found)
	return found
}

func checkUseCasesImportOnlyDomainAndPorts(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	imports, read := getUseCaseImports(load.GetModulePackages(t, workspace, "libs/core/kernel"))
	for _, one := range getBadUseCaseImports(imports, getHelperPackages()) {
		t.Error(one)
	}
	if read < 20 {
		t.Fatalf("%d use case packages read: the rule is checked against nothing", read)
	}
}
