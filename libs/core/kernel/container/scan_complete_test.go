package container

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// layerGroups are the folders of the kernel a package may live in. A package belongs to exactly one, so a
// new folder cannot sit outside every rule that reads a layer.
func listLayerGroups() []string {
	return []string{"domain", "port", "usecase", "adapter", "internal/adapter", "internal/adapterlib", "internal/testsupport", "container"}
}

// getHelperPackages are the pure packages that belong to no layer, each with the reason it needs none. A
// helper holds no scenario and no port: it computes over values, or it only hands out a grant. A package
// with a scenario or a port-like interface belongs in usecase/ or port/.
func getHelperPackages() map[string]string {
	return map[string]string{
		"internal/appearance": "builds the appearance section of a page from settings values",
		"internal/csp":        "builds the content security policy string of a window",
		"internal/operation":  "memo of what one operation learned, released when the operation ends",
		"internal/testonly":   "the grant that only tests may hold",
		"internal/text":       "byte-range arithmetic over text",
		"internal/vector":     "quantises a vector to bytes and bits",
	}
}

func getLayerGroup(rel string, helpers map[string]string) (group string, ok bool) {
	var found []string
	for _, one := range listLayerGroups() {
		if isWithin(rel, one) {
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

// listProductionDirs returns the directories under root that hold a production Go file, by slash path
// relative to root.
func listProductionDirs(root string) ([]string, error) {
	seen := map[string]bool{}
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
			seen[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	dirs := make([]string, 0, len(seen))
	for dir := range seen {
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

func TestEveryKernelPackageBelongsToOneKnownLayer(t *testing.T) {
	dirs, err := listProductionDirs("..")
	if err != nil {
		t.Fatal(err)
	}
	helpers := getHelperPackages()
	for _, one := range getUnplacedPackages(dirs, helpers) {
		t.Error(one)
	}

	// Every helper must exist, or its entry hides a package that is gone and may return unseen.
	for helper, reason := range helpers {
		if !slices.Contains(dirs, helper) || reason == "" {
			t.Errorf("the helper %s is not a package of the kernel or has no reason", helper)
		}
	}
	if len(dirs) < 100 {
		t.Fatalf("%d packages read: the walk is not reading the kernel", len(dirs))
	}
}

// A package outside every group fails, a package in a helper entry passes, and a layer folder that is
// only the prefix of another (adapter and adapterlib) does not take its neighbour.
func TestScanRule_RejectsAPackageOutsideEveryGroup(t *testing.T) {
	helpers := map[string]string{"internal/csp": "policy string"}
	got := getUnplacedPackages([]string{
		"domain/theme", "port", "port/noteport", "usecase/note/query", "adapter/window/editor", "internal/adapter/clock",
		"internal/adapterlib/wire", "internal/testsupport/indexfake", "container", "internal/csp",
		"internal/vaultreading", "internal/adapterx", "scratch", "internal/csp/sub",
	}, helpers)
	want := []string{
		`internal/vaultreading is in "" groups, want exactly one`,
		`internal/adapterx is in "" groups, want exactly one`,
		`scratch is in "" groups, want exactly one`,
		`internal/csp/sub is in "" groups, want exactly one`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

// getUseCaseImports lists the packages of the kernel that production files of use cases import.
func getUseCaseImports(pkgs []*packages.Package) (imports map[string][]string, read int) {
	imports = map[string][]string{}
	for _, pkg := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, coreImportPath), "/")
		if !isWithin(rel, "usecase") {
			continue
		}
		read++
		for path := range pkg.Imports {
			if isWithin(path, coreImportPath) {
				target := strings.TrimPrefix(strings.TrimPrefix(path, coreImportPath), "/")
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

func checkUseCasesImportOnlyDomainAndPorts(t *testing.T, workspace workspacePackages) {
	t.Helper()
	imports, read := getUseCaseImports(getModulePackages(t, workspace, "libs/core/kernel"))
	for _, one := range getBadUseCaseImports(imports, getHelperPackages()) {
		t.Error(one)
	}
	if read < 20 {
		t.Fatalf("%d use case packages read: the rule is checked against nothing", read)
	}
}

func TestUseCaseImportRule_RejectsASharedPackageOutsideDomainPortAndHelpers(t *testing.T) {
	helpers := map[string]string{"internal/text": "ranges"}
	got := getBadUseCaseImports(map[string][]string{
		"usecase/note": {"domain/theme", "port/noteport", "internal/text", "internal/vaultreading", "adapter/index", "usecase/note/query"},
	}, helpers)
	want := []string{
		"usecase/note imports adapter/index (adapter): a use case imports the domain, ports and its own helpers",
		`usecase/note imports internal/vaultreading (): a use case imports the domain, ports and its own helpers`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}
