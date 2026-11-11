package plugingomod

import (
	"os"
	"path/filepath"
	"testing"
	"golang.org/x/mod/modfile"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// getPluginGoModViolations defines the whole rule, so it is tested directly.
func TestWhatThePluginGoModRulesReject(t *testing.T) {
	const head = "module " + load.PluginsPrefix + "sample\n\ngo 1.26\n\n"
	for _, one := range []struct {
		name, text string
		rejected   int
	}{
		{"the SDK and third-party code", head + "require (\n\t" + load.SdkModule + " v0.0.0\n\t" + load.SdkModule + "/host v0.0.0\n\tgolang.org/x/net v0.1.0\n)\n", 0},
		{"the notes library as Go records it for the SDK", head + "require " + load.NotesModule + " v0.0.0 // indirect\n", 0},
		{"the RPC library as Go records it for the SDK", head + "require connectrpc.com/connect v1.0.0 // indirect\n", 0},
		{"a direct requirement of the notes library", head + "require " + load.NotesModule + " v0.0.0\n", 1},
		{"a direct requirement of the protocol", head + "require (\n\t" + load.Wire + " v0.0.0\n)\n", 1},
		{"a direct requirement of the RPC library", head + "require connectrpc.com/connect v1.0.0\n", 1},
		{"a direct requirement of protobuf", head + "require google.golang.org/protobuf v1.0.0\n", 1},
		{"a direct requirement of another plugin", head + "require " + load.PluginsPrefix + "other v0.0.0\n", 1},
		{"a direct requirement of the core", head + "require " + load.Module + "port v0.0.0\n", 1},
		{"a module built from a folder", head + "require example.com/x v1.0.0\nreplace example.com/x => ../x\n", 1},
		{"a module replaced by a published one", head + "require example.com/x v1.0.0\nreplace example.com/x => example.com/y v1.0.0\n", 0},
		{"another module path", "module " + load.PluginsPrefix + "other\n", 1},
	} {
		file, err := modfile.Parse("go.mod", []byte(one.text), nil)
		if err != nil {
			t.Fatalf("%s: %v", one.name, err)
		}
		if got := getPluginGoModViolations("sample", file); len(got) != one.rejected {
			t.Errorf("%s: violations %v, want %d", one.name, got, one.rejected)
		}
	}
}

// The go.mod of every plugin declares the module of its folder and requires directly only the
// plugin SDK and third-party code.
func TestNoPluginModuleRequiresMoreThanTheSDK(t *testing.T) {
	for _, dir := range load.ListPluginModules(t) {
		file := load.ReadGoMod(t, filepath.Join(dir, "go.mod"))
		for _, why := range getPluginGoModViolations(filepath.Base(dir), file) {
			t.Errorf("%s/go.mod %s", dir, why)
		}
	}
}
