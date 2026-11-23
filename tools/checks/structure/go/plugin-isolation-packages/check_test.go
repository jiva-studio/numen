// Package pluginisolationpackages is the check `go/plugin-isolation-packages`, run on the tree with the planted violations that prove it.
package pluginisolationpackages

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
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
	sharedLib := &packages.Package{ID: "shared", PkgPath: load.NumenRoot + "libs/shared/tokenizer", Imports: map[string]*packages.Package{"x": {ID: "uniseg", PkgPath: "github.com/rivo/uniseg"}}}
	atomicfile := &packages.Package{ID: "atomicfile", PkgPath: load.NumenRoot + "libs/shared/atomicfile"}
	wire := &packages.Package{ID: "wire", PkgPath: load.Wire + "/gen/numen/v1"}
	wireViaSDK := &packages.Package{ID: "wirevia", PkgPath: load.SdkModule + "/wirevia", Imports: map[string]*packages.Package{"x": wire}}
	sharedLookalike := &packages.Package{ID: "lookalike", PkgPath: load.NumenRoot + "libs/sharedx/onnx"}
	for name, one := range map[string]struct {
		imports map[string]*packages.Package
		want    []string
	}{
		"own and sdk packages":                 {map[string]*packages.Package{"a": own, "b": sdk}, nil},
		"another plugin":                       {map[string]*packages.Package{"a": other}, []string{other.PkgPath}},
		"the core":                             {map[string]*packages.Package{"a": core}, []string{core.PkgPath}},
		"the notes library":                    {map[string]*packages.Package{"a": notes}, []string{notes.PkgPath}},
		"the protocol":                         {map[string]*packages.Package{"a": wire}, []string{wire.PkgPath}},
		"the protocol through the sdk":         {map[string]*packages.Package{"a": wireViaSDK}, nil},
		"the core through the sdk":             {map[string]*packages.Package{"a": viaSDK}, []string{core.PkgPath}},
		"notes through the sdk":                {map[string]*packages.Package{"a": notesViaSDK}, nil},
		"a shared library":                     {map[string]*packages.Package{"a": sharedLib}, nil},
		"the atomic file library":              {map[string]*packages.Package{"a": atomicfile}, nil},
		"a library named like the shared ones": {map[string]*packages.Package{"a": sharedLookalike}, []string{sharedLookalike.PkgPath}},
	} {
		root := &packages.Package{ID: "root", PkgPath: load.PluginsPrefix + "sample/backend", Imports: one.imports}
		got := getForeignDependencies(root)
		if !slices.Equal(got, one.want) {
			t.Errorf("%s: foreign dependencies %v, want %v", name, got, one.want)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
