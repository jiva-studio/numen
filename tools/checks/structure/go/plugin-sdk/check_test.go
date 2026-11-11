// Package pluginsdk is the check `go/plugin-sdk`, run on the tree with the planted violations that prove it.
package pluginsdk

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// getSDKRefusal defines the whole rule, so it is tested directly.
func TestWhatTheSDKModuleRulesRefuse(t *testing.T) {
	for _, one := range []struct {
		mod, dep  string
		isRefused bool
	}{
		{load.SdkModule, load.SdkModule + "/onnx", true},
		{load.SdkModule, load.SdkModule + "/agent", true},
		{load.SdkModule + "/onnx", load.SdkModule + "/models", true},
		{load.SdkModule + "/agent", load.SdkModule + "/vault", true},
		{load.SdkModule + "/models", load.SdkModule, false},
		{load.SdkModule + "/agent", load.Wire, false},
		{load.SdkModule, load.Wire, false},
		{load.SdkModule, "connectrpc.com/connect", false},
		{load.SdkModule + "/onnx", "github.com/getcharzp/onnxruntime_purego", false},
	} {
		if got := getSDKRefusal(one.mod, one.dep) != ""; got != one.isRefused {
			t.Errorf("%s requires %s: failure is %v, want %v", one.mod, one.dep, got, one.isRefused)
		}
	}
	if load.IsSDK(load.NumenRoot + "libs/plugin/sdk-ts") {
		t.Error("the TypeScript SDK is taken for a module of the Go SDK")
	}
}

// Every go.mod of the plugin SDK is named after its folder and requires only
// what getSDKRefusal lets it.
func TestNoSDKModuleRequiresAFeatureModule(t *testing.T) {
	for _, one := range findSDKModuleFindings(t, filepath.Join("..", "..", "..", "plugin", "sdk-go")) {
		t.Error(one)
	}
}

// TestSDKModuleWalk_FindsThePlantedModules verifies that the walk finds a module named after another
// folder and a module that requires a feature module of the SDK, and passes a correct one.
func TestSDKModuleWalk_FindsThePlantedModules(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"go.mod":           "module " + load.SdkModule + "\n\ngo 1.26\n",
		"sdk/vault/go.mod": "module " + load.SdkModule + "/wrong\n\ngo 1.26\n",
		"sdk/onnx/go.mod":  "module " + load.SdkModule + "/onnx\n\ngo 1.26\n\nrequire " + load.SdkModule + "/agent v0.0.0\n",
	})
	found := findSDKModuleFindings(t, filepath.Join(root, "sdk"))
	if len(found) != 2 {
		t.Errorf("the walk found %v, want the wrong name of vault and the requirement of onnx", found)
	}
}

// Fails the test when it reads no go.mod file.
func findSDKModuleFindings(t *testing.T, root string) []string {
	t.Helper()
	var wrong []string
	found := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() != "go.mod" {
			return err
		}
		found++
		file := load.ReadGoMod(t, path)
		dir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		want := load.SdkModule
		if dir != "." {
			want += "/" + filepath.ToSlash(dir)
		}
		mod := load.GetModulePath(file)
		if mod != want {
			wrong = append(wrong, fmt.Sprintf("%s declares module %q, want %q", path, mod, want))
		}
		for _, required := range file.Require {
			if why := getSDKRefusal(mod, required.Mod.Path); why != "" {
				wrong = append(wrong, fmt.Sprintf("%s requires %s: %s", mod, required.Mod.Path, why))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found == 0 {
		t.Fatalf("no go.mod found under %s: a rule checked against nothing", root)
	}
	return wrong
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
