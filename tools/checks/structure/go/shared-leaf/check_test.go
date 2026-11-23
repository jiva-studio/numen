// Package sharedleaf is the check `go/shared-leaf`, run on the tree with the planted violations that prove it.
package sharedleaf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/mod/modfile"
)

func TestWhatTheSharedLeafRuleRejects(t *testing.T) {
	const head = "module " + load.SharedLibraries + "/sample\n\ngo 1.26\n\n"
	for _, one := range []struct {
		name, text string
		rejected   int
	}{
		{"third-party code", head + "require golang.org/x/net v0.1.0\n", 0},
		{"another shared library", head + "require " + load.SharedLibraries + "/other v0.0.0\n", 0},
		{"an indirect requirement of third-party code", head + "require golang.org/x/net v0.1.0 // indirect\n", 0},
		{"a direct requirement of a core module", head + "require " + load.Module + "port v0.0.0\n", 1},
		{"an indirect requirement of a core module", head + "require " + load.Module + "port v0.0.0 // indirect\n", 1},
		{"the protocol, which lies in libs/shared", head + "require " + load.Wire + " v0.0.0\n", 0},
		{"a direct requirement of the plugin SDK", head + "require " + load.SdkModule + " v0.0.0\n", 1},
		{"a direct requirement of a plugin", head + "require " + load.PluginsPrefix + "other v0.0.0\n", 1},
		{"a library named like the shared ones", head + "require " + load.SharedLibraries + "x v0.0.0\n", 1},
		{"a shared library built from a folder of libs/shared", head + "require " + load.SharedLibraries + "/other v0.0.0\nreplace " + load.SharedLibraries + "/other => ../other\n", 0},
		{"a shared library built from a folder outside libs/shared", head + "require " + load.SharedLibraries + "/other v0.0.0\nreplace " + load.SharedLibraries + "/other => ../../core/other\n", 1},
		{"third-party code built from a folder outside the repository", head + "require example.com/x v1.0.0\nreplace example.com/x => ../../../../x\n", 1},
		{"third-party code replaced by a published module", head + "require example.com/x v1.0.0\nreplace example.com/x => example.com/y v1.0.0\n", 0},
	} {
		file, err := modfile.Parse("go.mod", []byte(one.text), nil)
		if err != nil {
			t.Fatalf("%s: %v", one.name, err)
		}
		if got := getSharedLeafViolations("libs/shared/sample", file); len(got) != one.rejected {
			t.Errorf("%s: violations %v, want %d", one.name, got, one.rejected)
		}
	}
}

// Every module under libs/shared requires, direct or indirect, only third-party modules and other libs/shared modules.
func TestNoSharedLibraryRequiresAModuleOutsideLibsShared(t *testing.T) {
	for _, dir := range load.ListModulesBelow(t, "libs/shared") {
		file := load.ReadGoMod(t, filepath.Join(load.RepoRootDir, dir, "go.mod"))
		for _, why := range getSharedLeafViolations(dir, file) {
			t.Errorf("%s/go.mod %s", dir, why)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
