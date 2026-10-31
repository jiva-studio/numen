package container

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// pluginCapture matches the package path of a plugin and captures the plugin directory name.
var pluginCapture = regexp.MustCompile(`^` + regexp.QuoteMeta(pluginsPrefix) + `([^/]+?)(?:\.test|_test)?(?:/|$)`)

// getPluginCapture returns the plugin directory a package path belongs to, or "" for any other package.
func getPluginCapture(pkgPath string) string {
	if match := pluginCapture.FindStringSubmatch(pkgPath); match != nil {
		return match[1]
	}
	return ""
}

// isForeignToPlugins reports whether a package may be reached by no plugin, directly or through another
// package: another plugin, the core or an application.
func isForeignToPlugins(own, pkgPath string) bool {
	if one := getPluginCapture(pkgPath); one != "" {
		return one != own
	}
	return pkgPath == strings.TrimSuffix(module, "/") || strings.HasPrefix(pkgPath, module) || strings.HasPrefix(pkgPath, numenRoot+"apps/")
}

// getForeignDependencies returns every package in the import closure of root that the plugin may not
// reach. The packages root imports directly answer to the whole dependency rule of a plugin; the
// packages beyond them are the dependencies of the SDK and the protocol, which may use the notes library.
func getForeignDependencies(root *packages.Package) []string {
	own := getPluginCapture(root.PkgPath)
	visited := map[string]bool{}
	var foreign []string
	var walk func(pkg *packages.Package, isDirect bool)
	walk = func(pkg *packages.Package, isDirect bool) {
		if visited[pkg.ID] {
			return
		}
		visited[pkg.ID] = true
		if isForeignToPlugins(own, pkg.PkgPath) || (isDirect && getPluginCapture(pkg.PkgPath) != own && getPluginDependencyRefusal(pluginsPrefix+own, pkg.PkgPath) != "") {
			foreign = append(foreign, pkg.PkgPath)
		}
		for _, imported := range pkg.Imports {
			walk(imported, false)
		}
	}
	for _, imported := range root.Imports {
		walk(imported, true)
	}
	slices.Sort(foreign)
	return slices.Compact(foreign)
}

// The type-checked import graph of a plugin, tests included, reaches no other plugin and no package
// of the core, the notes library or an application, directly or through another package.
func TestNoPluginReachesAnotherPluginThroughTheImportGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("loads the packages of every plugin module")
	}
	dirs := listPluginModules(t)
	loaded := 0
	for _, dir := range dirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			t.Fatal(err)
		}
		config := &packages.Config{
			Dir:   abs,
			Tests: true,
			Mode:  packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedFiles,
		}
		roots, err := packages.Load(config, "./...")
		if err != nil {
			t.Fatal(err)
		}
		for _, root := range roots {
			own := getPluginCapture(root.PkgPath)
			if own == "" {
				continue
			}
			loaded++
			if own != filepath.Base(dir) {
				t.Errorf("%s holds the package %s of plugin %s", dir, root.PkgPath, own)
			}
			for _, dep := range getForeignDependencies(root) {
				t.Errorf("%s reaches %s", root.PkgPath, dep)
			}
		}
	}
	if loaded < len(dirs) {
		t.Fatalf("%d plugin packages loaded for %d plugin modules: a rule checked against nothing", loaded, len(dirs))
	}
}

// getForeignDependencies defines the whole rule, so it is tested on a synthetic import graph.
func TestPluginIsolationRule_RejectsForeignPackagesInTheClosure(t *testing.T) {
	sdk := &packages.Package{ID: "sdk", PkgPath: sdkModule + "/vault"}
	other := &packages.Package{ID: "other", PkgPath: pluginsPrefix + "other/backend/service"}
	core := &packages.Package{ID: "core", PkgPath: module + "port/pluginport"}
	notes := &packages.Package{ID: "notes", PkgPath: notesModule + "/domain"}
	own := &packages.Package{ID: "own", PkgPath: pluginsPrefix + "sample/backend/service"}
	viaSDK := &packages.Package{ID: "via", PkgPath: sdkModule + "/via", Imports: map[string]*packages.Package{"x": core}}
	notesViaSDK := &packages.Package{ID: "notesvia", PkgPath: sdkModule + "/notesvia", Imports: map[string]*packages.Package{"x": notes}}
	for name, one := range map[string]struct {
		imports map[string]*packages.Package
		want    []string
	}{
		"own and sdk packages":     {map[string]*packages.Package{"a": own, "b": sdk}, nil},
		"another plugin":           {map[string]*packages.Package{"a": other}, []string{other.PkgPath}},
		"the core":                 {map[string]*packages.Package{"a": core}, []string{core.PkgPath}},
		"the notes library":        {map[string]*packages.Package{"a": notes}, []string{notes.PkgPath}},
		"the core through the sdk": {map[string]*packages.Package{"a": viaSDK}, []string{core.PkgPath}},
		"notes through the sdk":    {map[string]*packages.Package{"a": notesViaSDK}, nil},
	} {
		root := &packages.Package{ID: "root", PkgPath: pluginsPrefix + "sample/backend", Imports: one.imports}
		got := getForeignDependencies(root)
		if !slices.Equal(got, one.want) {
			t.Errorf("%s: foreign dependencies %v, want %v", name, got, one.want)
		}
	}
}
