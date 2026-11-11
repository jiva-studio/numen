package pluginisolationpackages

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The type-checked import graph of a plugin, tests included, reaches no other plugin and no package
// of the core, the notes library or an application, directly or through another package.
func TestNoPluginReachesAnotherPluginThroughTheImportGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("loads the packages of every plugin module")
	}
	dirs := load.ListPluginModules(t)
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
	sdk := &packages.Package{ID: "sdk", PkgPath: load.SdkModule + "/vault"}
	other := &packages.Package{ID: "other", PkgPath: load.PluginsPrefix + "other/backend/service"}
	core := &packages.Package{ID: "core", PkgPath: load.Module + "port/pluginport"}
	notes := &packages.Package{ID: "notes", PkgPath: load.NotesModule + "/domain"}
	own := &packages.Package{ID: "own", PkgPath: load.PluginsPrefix + "sample/backend/service"}
	viaSDK := &packages.Package{ID: "via", PkgPath: load.SdkModule + "/via", Imports: map[string]*packages.Package{"x": core}}
	notesViaSDK := &packages.Package{ID: "notesvia", PkgPath: load.SdkModule + "/notesvia", Imports: map[string]*packages.Package{"x": notes}}
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
		root := &packages.Package{ID: "root", PkgPath: load.PluginsPrefix + "sample/backend", Imports: one.imports}
		got := getForeignDependencies(root)
		if !slices.Equal(got, one.want) {
			t.Errorf("%s: foreign dependencies %v, want %v", name, got, one.want)
		}
	}
}
