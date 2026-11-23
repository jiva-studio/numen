// Package extractablerequirements is the check `go/extractable-requirements`, run on the tree with the planted violations that prove it.
package extractablerequirements

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/mod/modfile"
)

func TestWhatTheExtractableRequirementsRuleRejects(t *testing.T) {
	const head = "module " + load.SdkModule + "/host\n\ngo 1.26\n\n"
	for _, one := range []struct {
		name, text string
		rejected   int
	}{
		{"third-party code", head + "require golang.org/x/net v0.1.0\n", 0},
		{"the plugin SDK", head + "require " + load.SdkModule + " v0.0.0\n", 0},
		{"a shared library", head + "require " + load.SharedLibraries + "/onnx v0.0.0\n", 0},
		{"the notes library", head + "require " + load.NotesModule + " v0.0.0\n", 0},
		{"an indirect requirement of the notes library", head + "require " + load.NotesModule + " v0.0.0 // indirect\n", 0},
		{"an indirect requirement of the protocol", head + "require " + load.Wire + " v0.0.0 // indirect\n", 0},
		{"an indirect requirement of a core module", head + "require " + load.Module + "port v0.0.0 // indirect\n", 1},
		{"the atomic file library", head + "require " + load.NumenRoot + "libs/shared/atomicfile v0.0.0\n", 0},
		{"a direct requirement of the plugin catalog", head + "require " + load.NumenRoot + "libs/plugin/catalog v0.0.0\n", 1},
		{"a direct requirement of the core", head + "require " + load.Module + "port v0.0.0\n", 1},
		{"a direct requirement of a plugin", head + "require " + load.PluginsPrefix + "other v0.0.0\n", 1},
		{"a library named like the shared ones", head + "require " + load.SharedLibraries + "x v0.0.0\n", 1},
		{"a library named like the SDK", head + "require " + load.SdkModule + "x v0.0.0\n", 1},
		{"the SDK built from its folder", head + "require " + load.SdkModule + " v0.0.0\nreplace " + load.SdkModule + " => ../\n", 0},
		{"the SDK built from a folder outside the set", head + "require " + load.SdkModule + " v0.0.0\nreplace " + load.SdkModule + " => ../../../core/sdk\n", 1},
		{"the notes library built from a folder outside libs/shared", head + "require " + load.NotesModule + " v0.0.0\nreplace " + load.NotesModule + " => ../../../core/other\n", 1},
		{"third-party code built from a folder of the repository outside the set", head + "require example.com/x v1.0.0\nreplace example.com/x => ../../../core/other\n", 1},
		{"third-party code replaced by a published module", head + "require example.com/x v1.0.0\nreplace example.com/x => example.com/y v1.0.0\n", 0},
	} {
		file, err := modfile.Parse("go.mod", []byte(one.text), nil)
		if err != nil {
			t.Fatalf("%s: %v", one.name, err)
		}
		if got := getExtractableViolations("libs/plugin/sdk-go/host", file); len(got) != one.rejected {
			t.Errorf("%s: violations %v, want %d", one.name, got, one.rejected)
		}
	}
}

func TestAPluginRequiresNoPluginAndItsOwnModuleIsAllowed(t *testing.T) {
	file, err := modfile.Parse("go.mod", []byte("module "+load.PluginsPrefix+"sample\n\nrequire "+load.PluginsPrefix+"sample/sub v0.0.0\nrequire "+load.PluginsPrefix+"other v0.0.0\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := getExtractableViolations("plugins/sample", file); len(got) != 1 {
		t.Errorf("violations %v, want only the other plugin", got)
	}
}

// The modules of the plugin SDK and of the plugins require, direct or indirect, only third-party modules, the
// plugin SDK and libs/shared.
func TestNoExtractableModuleRequiresAModuleOutsideTheSet(t *testing.T) {
	dirs := append(load.ListModulesBelow(t, "libs/plugin/sdk-go"), load.GetPluginModuleDirs(t)...)
	for _, dir := range dirs {
		file := load.ReadGoMod(t, filepath.Join(load.RepoRootDir, dir, "go.mod"))
		for _, why := range getExtractableViolations(dir, file) {
			t.Errorf("%s/go.mod %s", dir, why)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
