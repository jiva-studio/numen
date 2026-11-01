package container

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/mod/modfile"
)

// readGoMod parses the go.mod at path.
func readGoMod(t *testing.T, path string) *modfile.File {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := modfile.Parse(path, text, nil)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

// getModulePath returns the path a go.mod declares in its module directive, or "" when it has none.
func getModulePath(file *modfile.File) string {
	if file.Module == nil {
		return ""
	}
	return file.Module.Mod.Path
}

// getPluginGoModRefusals returns why the go.mod of the plugin in the folder named dir is refused: a
// module path other than the one of the folder, a direct requirement getPluginDependencyRefusal
// refuses, or a module built from a local folder. An indirect requirement is what Go records of
// the requirements of the SDK and is allowed.
func getPluginGoModRefusals(dir string, file *modfile.File) []string {
	var refusals []string
	mod := getModulePath(file)
	if want := pluginsPrefix + dir; mod != want {
		refusals = append(refusals, "declares module "+mod+", want "+want)
	}
	for _, required := range file.Require {
		if required.Indirect {
			continue
		}
		if why := getPluginDependencyRefusal(mod, required.Mod.Path); why != "" {
			refusals = append(refusals, "requires "+required.Mod.Path+": "+why)
		}
	}
	for _, replaced := range file.Replace {
		if replaced.New.Version == "" {
			refusals = append(refusals, "builds "+replaced.Old.Path+" from the folder "+replaced.New.Path+
				": a plugin builds in a repository of its own")
		}
	}
	return refusals
}

// getPluginGoModRefusals defines the whole rule, so it is tested directly.
func TestWhatThePluginGoModRulesRefuse(t *testing.T) {
	const head = "module " + pluginsPrefix + "sample\n\ngo 1.26\n\n"
	for _, one := range []struct {
		name, text string
		refused    int
	}{
		{"the SDK and third-party code", head + "require (\n\t" + sdkModule + " v0.0.0\n\t" + sdkModule + "/host v0.0.0\n\tgolang.org/x/net v0.1.0\n)\n", 0},
		{"the notes library as Go records it for the SDK", head + "require " + notesModule + " v0.0.0 // indirect\n", 0},
		{"the RPC library as Go records it for the SDK", head + "require connectrpc.com/connect v1.0.0 // indirect\n", 0},
		{"a direct requirement of the notes library", head + "require " + notesModule + " v0.0.0\n", 1},
		{"a direct requirement of the protocol", head + "require (\n\t" + wire + " v0.0.0\n)\n", 1},
		{"a direct requirement of the RPC library", head + "require connectrpc.com/connect v1.0.0\n", 1},
		{"a direct requirement of protobuf", head + "require google.golang.org/protobuf v1.0.0\n", 1},
		{"a direct requirement of another plugin", head + "require " + pluginsPrefix + "other v0.0.0\n", 1},
		{"a direct requirement of the core", head + "require " + module + "port v0.0.0\n", 1},
		{"a module built from a folder", head + "require example.com/x v1.0.0\nreplace example.com/x => ../x\n", 1},
		{"a module replaced by a published one", head + "require example.com/x v1.0.0\nreplace example.com/x => example.com/y v1.0.0\n", 0},
		{"another module path", "module " + pluginsPrefix + "other\n", 1},
	} {
		file, err := modfile.Parse("go.mod", []byte(one.text), nil)
		if err != nil {
			t.Fatalf("%s: %v", one.name, err)
		}
		if got := getPluginGoModRefusals("sample", file); len(got) != one.refused {
			t.Errorf("%s: refusals %v, want %d", one.name, got, one.refused)
		}
	}
}

// The go.mod of every plugin declares the module of its folder and requires directly only the
// plugin SDK and third-party code.
func TestNoPluginModuleRequiresMoreThanTheSDK(t *testing.T) {
	for _, dir := range listPluginModules(t) {
		file := readGoMod(t, filepath.Join(dir, "go.mod"))
		for _, why := range getPluginGoModRefusals(filepath.Base(dir), file) {
			t.Errorf("%s/go.mod %s", dir, why)
		}
	}
}
