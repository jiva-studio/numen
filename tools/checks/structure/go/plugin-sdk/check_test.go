package pluginsdk

import (
	"io/fs"
	"path/filepath"
	"strings"
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
	root := filepath.Join("..", "..", "..", "plugin", "sdk-go")
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
			t.Errorf("%s declares module %q, want %q", path, mod, want)
		}
		for _, required := range file.Require {
			if why := getSDKRefusal(mod, required.Mod.Path); why != "" {
				t.Errorf("%s requires %s: %s", mod, required.Mod.Path, why)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found < 2 {
		t.Fatalf("%d go.mod files found under %s: a rule checked against nothing", found, root)
	}
}
