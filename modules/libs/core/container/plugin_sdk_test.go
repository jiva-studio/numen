package container

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sdkModule is the base module of the plugin SDK. Each feature module is a
// folder under it with a go.mod of its own, named after that folder.
const sdkModule = numenModules + "libs/plugin-sdk-go"

// isSDK says whether a package or module is part of the plugin SDK.
func isSDK(path string) bool {
	return path == sdkModule || strings.HasPrefix(path, sdkModule+"/")
}

// getSDKRefusal says why the SDK module mod may not require dep, or "" when it
// may. The base module requires no feature module, and a feature module
// requires the base module at most.
func getSDKRefusal(mod, dep string) string {
	switch {
	case !isSDK(dep), dep == mod:
		return ""
	case mod == sdkModule:
		return "the base module requires no feature module"
	case dep != sdkModule:
		return "a feature module requires no other feature module"
	}
	return ""
}

// What getSDKRefusal says is the whole of the rule, so it is asked directly.
func TestWhatTheSDKModuleRulesRefuse(t *testing.T) {
	for _, one := range []struct {
		mod, dep  string
		isRefused bool
	}{
		{sdkModule, sdkModule + "/onnx", true},
		{sdkModule, sdkModule + "/agent", true},
		{sdkModule + "/onnx", sdkModule + "/models", true},
		{sdkModule + "/agent", sdkModule + "/vault", true},
		{sdkModule + "/models", sdkModule, false},
		{sdkModule + "/agent", wire, false},
		{sdkModule, wire, false},
		{sdkModule, "connectrpc.com/connect", false},
		{sdkModule + "/onnx", "github.com/getcharzp/onnxruntime_purego", false},
	} {
		if got := getSDKRefusal(one.mod, one.dep) != ""; got != one.isRefused {
			t.Errorf("%s requires %s: refused is %v, want %v", one.mod, one.dep, got, one.isRefused)
		}
	}
	if isSDK(numenModules + "libs/plugin-sdk") {
		t.Error("the TypeScript SDK is taken for a module of the Go SDK")
	}
}

// getModulePath is the path a go.mod declares in its module directive.
func getModulePath(gomod string) string {
	for line := range strings.SplitSeq(gomod, "\n") {
		if path, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.Trim(strings.TrimSpace(path), `"`)
		}
	}
	return ""
}

// Every go.mod of the plugin SDK is named after its folder and requires only
// what getSDKRefusal lets it.
func TestNoSDKModuleRequiresAFeatureModule(t *testing.T) {
	root := filepath.Join("..", "..", "plugin-sdk-go")
	found := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() != "go.mod" {
			return err
		}
		found++
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		want := sdkModule
		if dir != "." {
			want += "/" + filepath.ToSlash(dir)
		}
		mod := getModulePath(string(text))
		if mod != want {
			t.Errorf("%s declares module %q, want %q", path, mod, want)
		}
		for _, dep := range parseGoMod(string(text)).requires {
			if why := getSDKRefusal(mod, dep); why != "" {
				t.Errorf("%s requires %s: %s", mod, dep, why)
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
